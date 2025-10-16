# Project code dump

- Generated: 2025-10-16 16:36:34+0300
- Root: `/home/ilya/projects/social_network_system_design/services/media-service`

cmd/app/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"media-service/internal/media"
	"media-service/internal/shared/httpx"
	"media-service/internal/storage/s3"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func initOTEL(ctx context.Context) func(context.Context) error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4318"
	}
	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("otel exporter: %v", err)
	}
	res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(os.Getenv("OTEL_SERVICE_NAME")),
		attribute.String("deployment.environment", "local"),
	))
	tp := trace.NewTracerProvider(trace.WithBatcher(exp), trace.WithResource(res))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	return tp.Shutdown
}

func main() {
	ctx := context.Background()
	shutdown := initOTEL(ctx)
	defer func() {
		c, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = shutdown(c)
	}()

	s3cfg := s3.Config{
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		UseSSL:    false,
		Bucket:    envOr("S3_BUCKET", "media"),
	}
	store, err := s3.New(s3cfg)
	if err != nil {
		log.Fatalf("s3: %v", err)
	}
	if err := store.EnsureBucket(ctx); err != nil {
		log.Fatalf("s3 ensure bucket: %v", err)
	}

	svc := media.NewService(store)
	h := media.NewHandler(svc)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.Handle("GET /media/{key}", otelhttp.NewHandler(http.HandlerFunc(h.RedirectToSignedGet), "media.get"))

	protected := func(pattern string, handler http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(handler))
	}
	protected("POST /media/upload", otelhttp.NewHandler(http.HandlerFunc(h.Upload), "media.upload"))
	protected("DELETE /media/{key}", otelhttp.NewHandler(http.HandlerFunc(h.Delete), "media.delete"))
	protected("POST /media/presign", otelhttp.NewHandler(http.HandlerFunc(h.PresignPut), "media.presign"))

	addr := envOr("APP_PORT", ":8088")
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	log.Printf("media-service listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

internal/media/handler.go
package media

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"media-service/internal/shared/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	uid, _ := httpx.UserFromCtx(r) // optional
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "file required"}, http.StatusBadRequest)
		return
	}
	defer file.Close()

	prefix := r.FormValue("prefix")
	key := h.svc.BuildKey(prefix, header.Filename, uid)
	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	b, _ := io.ReadAll(file)
	if err := h.svc.s3.Put(r.Context(), key, ct, b); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	url, _ := h.svc.s3.PresignGet(r.Context(), key, 15*time.Minute)
	httpx.WriteJSON(w, map[string]any{
		"key":         key,
		"contentType": ct,
		"url":         url.String(),
	}, http.StatusCreated)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httpx.WriteJSON(w, map[string]any{"error": "missing key"}, http.StatusBadRequest)
		return
	}
	if err := h.svc.s3.Remove(r.Context(), key); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RedirectToSignedGet(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.NotFound(w, r)
		return
	}
	ttl := 5 * time.Minute
	if s := r.URL.Query().Get("ttl"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 3600 {
			ttl = time.Duration(n) * time.Second
		}
	}
	u, err := h.svc.s3.PresignGet(r.Context(), key, ttl)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}

func (h *Handler) PresignPut(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Key         string `json:"key"`
		Prefix      string `json:"prefix"`
		FileName    string `json:"file_name"`
		ContentType string `json:"content_type"`
		ExpirySec   int    `json:"expiry_seconds"`
	}
	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "invalid json"}, http.StatusBadRequest)
		return
	}
	if body.Key == "" {
		uid, _ := httpx.UserFromCtx(r)
		body.Key = h.svc.BuildKey(strings.Trim(body.Prefix, "/"), body.FileName, uid)
	}
	if body.ExpirySec <= 0 || body.ExpirySec > 3600 {
		body.ExpirySec = 900
	}
	u, err := h.svc.s3.PresignPut(r.Context(), body.Key, time.Duration(body.ExpirySec)*time.Second, body.ContentType)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	httpx.WriteJSON(w, map[string]any{
		"key":    body.Key,
		"url":    u.String(),
		"ttl":    body.ExpirySec,
		"method": "PUT",
	}, http.StatusOK)
}

internal/media/service.go
package media

import (
	"fmt"
	"path"
	"strings"
	"time"

	"media-service/internal/storage/s3"
)

type Service struct {
	s3 *s3.Storage
}

func NewService(s *s3.Storage) *Service { return &Service{s3: s} }

func (s *Service) BuildKey(prefix, filename string, userID string) string {
	fn := path.Base(filename)
	now := time.Now().UTC().Format("20060102T150405")
	p := strings.Trim(prefix, "/")
	if p != "" {
		return fmt.Sprintf("%s/%s_%s_%s", p, userID, now, fn)
	}
	return fmt.Sprintf("%s_%s_%s", userID, now, fn)
}

internal/shared/httpx/httpx.go
package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const userKey ctxKey = "uid"

func WriteJSON(w http.ResponseWriter, v any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func AuthMiddleware(next http.Handler) http.Handler {
	secret := os.Getenv("JWT_SECRET")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret == "" {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, "0")))
			return
		}
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			WriteJSON(w, map[string]string{"error": "missing bearer token"}, http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(h, "Bearer ")
		parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !parsed.Valid {
			WriteJSON(w, map[string]string{"error": "invalid token"}, http.StatusUnauthorized)
			return
		}
		claims, _ := parsed.Claims.(jwt.MapClaims)
		sub, _ := claims["sub"].(string)
		ctx := context.WithValue(r.Context(), userKey, sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromCtx(r *http.Request) (string, error) {
	v, _ := r.Context().Value(userKey).(string)
	if v == "" {
		return "", nil
	}
	return v, nil
}

internal/storage/s3/s3.go
package s3

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
}

type Storage struct {
	cfg    Config
	client *minio.Client
}

func New(cfg Config) (*Storage, error) {
	cl, err := minio.New(strings.TrimPrefix(cfg.Endpoint, "http://"), &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{cfg: cfg, client: cl}, nil
}

func (s *Storage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.cfg.Bucket)
	if err != nil {
		return err
	}
	if !exists {
		return s.client.MakeBucket(ctx, s.cfg.Bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func (s *Storage) Put(ctx context.Context, key string, contentType string, data []byte) error {
	_, err := s.client.PutObject(ctx, s.cfg.Bucket, key,
		bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *Storage) FPut(ctx context.Context, key, path, contentType string) error {
	_, err := s.client.FPutObject(ctx, s.cfg.Bucket, key, path, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *Storage) Remove(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.cfg.Bucket, key, minio.RemoveObjectOptions{})
}

func (s *Storage) PresignGet(ctx context.Context, key string, ttl time.Duration) (*url.URL, error) {
	return s.client.PresignedGetObject(ctx, s.cfg.Bucket, key, ttl, nil)
}

func (s *Storage) PresignPut(ctx context.Context, key string, ttl time.Duration, contentType string) (*url.URL, error) {
	reqParams := make(url.Values)
	if contentType != "" {
		reqParams.Set("content-type", contentType)
	}
	return s.client.PresignedPutObject(ctx, s.cfg.Bucket, key, ttl)
}

