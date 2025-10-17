# Project code dump

- Generated: 2025-10-17 11:38:02+0300
- Root: `/home/spetsar/projects/social_network_system_design/services/user-service`

cmd/app/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"users-service/internal/interest"
	"users-service/internal/migrate"
	"users-service/internal/profile"
	"users-service/internal/shared/db"
	"users-service/internal/shared/httpx"
	"users-service/internal/social"
	"users-service/internal/user"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"gorm.io/plugin/opentelemetry/tracing"
)

func atoiDef(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func initOTEL(ctx context.Context) func(context.Context) error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4318"
	}
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	if err != nil {
		log.Fatalf("otel exporter: %v", err)
	}
	res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(os.Getenv("OTEL_SERVICE_NAME")),
		attribute.String("deployment.environment", os.Getenv("ENV")),
	))
	ratio := 1.0
	if s := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); s != "" {
		if f, e := strconv.ParseFloat(s, 64); e == nil && f >= 0 && f <= 1 {
			ratio = f
		}
	}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(ratio))),
		trace.WithBatcher(exp),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
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

	store := db.OpenFromEnv()
	_ = store.Base.Use(tracing.NewPlugin())

	if os.Getenv("AUTO_MIGRATE") == "true" {
		n := atoiDef(os.Getenv("NUM_SHARDS"), 1)
		for i := 0; i < n; i++ {
			if err := migrate.AutoMigrateAll(store, i); err != nil {
				log.Fatalf("migrate shard %d: %v", i, err)
			}
		}
	}

	userRepo := user.NewRepository(store)
	userSvc := user.NewService(userRepo)

	profileRepo := profile.NewRepository(store)
	profileSvc := profile.NewService(profileRepo)

	interestRepo := interest.NewRepository(store)
	interestSvc := interest.NewService(interestRepo)

	socialRepo := social.NewRepository(store, userRepo)
	socialSvc := social.NewService(socialRepo)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	uh := user.NewHandler(userSvc)
	mux.Handle("POST /users", httpx.Wrap(uh.Register))
	mux.Handle("POST /users/login", httpx.Wrap(uh.Login))
	mux.Handle("GET /users/{user_id}", httpx.Wrap(uh.GetByID))

	protect := func(pattern string, h http.Handler) {
		mux.Handle(pattern, httpx.AuthMiddleware(h))
	}

	protect("GET /whoami", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, sh, err := httpx.UserFromCtx(r)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusUnauthorized)
			return
		}
		httpx.WriteJSON(w, map[string]any{"user_id": uid, "shard_id": sh}, http.StatusOK)
	}))

	protect("GET /users", httpx.Wrap(uh.ListMine))

	ph := profile.NewHandler(profileSvc)
	protect("PUT /profile", httpx.Wrap(ph.Upsert))
	protect("GET /profile/{user_id}", httpx.Wrap(ph.GetPublic))

	ih := interest.NewHandler(interestSvc)
	protect("POST /interests", httpx.Wrap(ih.Create))
	protect("POST /interests/{interest_id}", httpx.Wrap(ih.Attach))
	protect("DELETE /interests/{interest_id}", httpx.Wrap(ih.Detach))
	protect("GET /interests", httpx.Wrap(ih.ListMine))

	sh := social.NewHandler(socialSvc)
	protect("POST /follow/{target_id}", httpx.Wrap(sh.Follow))
	protect("DELETE /follow/{target_id}", httpx.Wrap(sh.Unfollow))
	protect("GET /follow", httpx.Wrap(sh.ListFollowing))
	protect("POST /friends/{friend_id}", httpx.Wrap(sh.Befriend))
	protect("DELETE /friends/{friend_id}", httpx.Wrap(sh.Unfriend))
	protect("GET /friends", httpx.Wrap(sh.ListFriends))
	protect("POST /relationships", httpx.Wrap(sh.CreateRelationship))
	protect("DELETE /relationships", httpx.Wrap(sh.DeleteRelationship))
	protect("GET /relationships", httpx.Wrap(sh.ListRelationships))

	addr := os.Getenv("APP_PORT")
	if addr == "" {
		addr = ":8081"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           otelhttp.NewHandler(mux, "http.server"),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	log.Printf("user-service listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

internal/interest/handler.go
package interest

import (
	"net/http"
	"strconv"

	"users-service/internal/shared/httpx"
	"users-service/internal/shared/validate"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

type CreateReq struct {
	Name string `json:"name" validate:"required"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	_, shardID, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[CreateReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	it, err := h.svc.Create(shardID, in.Name)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"id": it.ID, "name": it.Name}, http.StatusCreated)
	return nil
}

func (h *Handler) Attach(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	id, _ := strconv.ParseUint(r.PathValue("interest_id"), 10, 64)
	if id == 0 {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Attach(uid, id); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) Detach(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	id, _ := strconv.ParseUint(r.PathValue("interest_id"), 10, 64)
	if id == 0 {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Detach(uid, id); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.List(uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

internal/interest/interest.go
package interest

type City struct {
	ID   uint64 `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex;size:120" json:"name"`
}
type Interest struct {
	ID   uint64 `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex;size:120" json:"name"`
}
type InterestUser struct {
	UserID     string `gorm:"primaryKey;size:64"`
	InterestID uint64 `gorm:"primaryKey"`
}

internal/interest/repository.go
package interest

import (
	"users-service/internal/shared/db"
	"users-service/internal/shared/shard"

	"gorm.io/gorm"
)

type Repository interface {
	// NEW:
	Create(shardID int, name string) (*Interest, error)

	Attach(uid string, interestID uint64) error
	Detach(uid string, interestID uint64) error
	List(uid string, limit, offset int) ([]Interest, error)
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(shardID int, name string) (*Interest, error) {
	in := &Interest{Name: name}
	if err := r.store.Write(shardID).FirstOrCreate(in, "name = ?", name).Error; err != nil {
		return nil, err
	}
	return in, nil
}

func (r *repo) Attach(uid string, interestID uint64) error {
	sh, _ := shard.Extract(uid)
	return r.store.Write(sh).FirstOrCreate(&InterestUser{UserID: uid, InterestID: interestID}).Error
}
func (r *repo) Detach(uid string, interestID uint64) error {
	sh, _ := shard.Extract(uid)
	return r.store.Write(sh).Delete(&InterestUser{}, "user_id=? AND interest_id=?", uid, interestID).Error
}
func (r *repo) List(uid string, limit, offset int) ([]Interest, error) {
	sh, _ := shard.Extract(uid)
	var ints []Interest
	err := r.store.Use(sh).
		Joins("JOIN interest_users iu ON iu.interest_id = interests.id AND iu.user_id = ?", uid).
		Model(&Interest{}).Limit(limit).Offset(offset).Find(&ints).Error
	return ints, err
}

var _ = gorm.ErrRecordNotFound

internal/interest/service.go
package interest

type Service interface {
	Create(shardID int, name string) (*Interest, error)

	Attach(uid string, interestID uint64) error
	Detach(uid string, interestID uint64) error
	List(uid string, limit, offset int) ([]Interest, error)
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Create(shardID int, name string) (*Interest, error) {
	return s.repo.Create(shardID, name)
}
func (s *service) Attach(uid string, interestID uint64) error { return s.repo.Attach(uid, interestID) }
func (s *service) Detach(uid string, interestID uint64) error { return s.repo.Detach(uid, interestID) }
func (s *service) List(uid string, limit, offset int) ([]Interest, error) {
	return s.repo.List(uid, limit, offset)
}

internal/migrate/migrate.go
package migrate

import (
	"users-service/internal/interest"
	"users-service/internal/profile"
	"users-service/internal/shared/db"
	"users-service/internal/social"
	"users-service/internal/user"
)

func AutoMigrateAll(store *db.Store, shardID int) error {
	return store.Write(shardID).AutoMigrate(
		&user.User{},
		&profile.Profile{},
		&interest.City{}, &interest.Interest{}, &interest.InterestUser{},
		&social.Follow{}, &social.Friend{}, &social.Relationship{},
	)
}

internal/profile/handler.go
package profile

import (
	"net/http"

	"users-service/internal/shared/httpx"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[UpsertReq](r)
	if err != nil {
		return err
	}
	if err := h.svc.Upsert(uid, in); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}
func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) error {
	uid := r.PathValue("user_id")
	p, err := h.svc.GetPublic(uid)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, p, http.StatusOK)
	return nil
}

internal/profile/profile.go
package profile

import "time"

type Profile struct {
	UserID      string         `gorm:"primaryKey;size:64" json:"user_id"`
	Description string         `json:"description"`
	CityID      uint64         `json:"city_id"`
	Education   map[string]any `gorm:"type:jsonb" json:"education"`
	Hobby       map[string]any `gorm:"type:jsonb" json:"hobby"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type UpsertReq struct {
	Description string         `json:"description"`
	CityID      uint64         `json:"city_id"`
	Education   map[string]any `json:"education"`
	Hobby       map[string]any `json:"hobby"`
}

internal/profile/repository.go
package profile

import (
	"users-service/internal/shared/db"
	"users-service/internal/shared/shard"

	"gorm.io/gorm/clause"
)

type Repository interface {
	Upsert(p *Profile) error
	GetPublic(userID string) (*Profile, error)
}
type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Upsert(p *Profile) error {
	sh, _ := shard.Extract(p.UserID)
	return r.store.Write(sh).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"description", "city_id", "education", "hobby", "updated_at"}),
	}).Create(p).Error
}
func (r *repo) GetPublic(uid string) (*Profile, error) {
	sh, _ := shard.Extract(uid)
	var p Profile
	if err := r.store.Use(sh).First(&p, "user_id = ?", uid).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

internal/profile/service.go
package profile

import "time"

type Service interface {
	Upsert(uid string, in UpsertReq) error
	GetPublic(uid string) (*Profile, error)
}
type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Upsert(uid string, in UpsertReq) error {
	return s.repo.Upsert(&Profile{
		UserID: uid, Description: in.Description, CityID: in.CityID,
		Education: in.Education, Hobby: in.Hobby, UpdatedAt: time.Now(),
	})
}
func (s *service) GetPublic(uid string) (*Profile, error) { return s.repo.GetPublic(uid) }

internal/shared/db/sharded.go
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

type ShardCfg struct {
	ID      int      `json:"id"`
	Writer  string   `json:"writer"`
	Readers []string `json:"readers,omitempty"`
}

type Store struct {
	Base   *gorm.DB
	shards map[int]ShardCfg
}

func (s *Store) Use(shardID int) *gorm.DB {
	return s.Base.Clauses(dbresolver.Use(fmt.Sprintf("shard%d", shardID)))
}
func (s *Store) Write(shardID int) *gorm.DB {
	return s.Base.Clauses(
		dbresolver.Use(fmt.Sprintf("shard%d", shardID)),
		dbresolver.Write,
	)
}
func (s *Store) ShardInfo(id int) (ShardCfg, bool) { c, ok := s.shards[id]; return c, ok }

func OpenFromEnv() *Store {
	raw := os.Getenv("SHARDS_JSON")
	if raw == "" {
		log.Fatal("SHARDS_JSON not set")
	}

	var shards []ShardCfg
	if err := json.Unmarshal([]byte(raw), &shards); err != nil || len(shards) == 0 {
		log.Fatalf("invalid SHARDS_JSON: %v", err)
	}

	base, err := openWithRetry(shards[0].Writer, 8, 2*time.Second)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}

	sqlDB, _ := base.DB()
	sqlDB.SetMaxOpenConns(40)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	makeCfg := func(s ShardCfg) dbresolver.Config {
		var readers []gorm.Dialector
		for _, r := range s.Readers {
			readers = append(readers, postgres.Open(r))
		}
		return dbresolver.Config{
			Sources:  []gorm.Dialector{postgres.Open(s.Writer)},
			Replicas: readers,
			Policy:   dbresolver.RandomPolicy{},
		}
	}

	r := dbresolver.Register(makeCfg(shards[0]), fmt.Sprintf("shard%d", shards[0].ID))
	for _, s := range shards[1:] {
		r = r.Register(makeCfg(s), fmt.Sprintf("shard%d", s.ID))
	}
	if err := base.Use(r); err != nil {
		log.Fatalf("dbresolver: %v", err)
	}

	imap := make(map[int]ShardCfg, len(shards))
	for _, s := range shards {
		imap[s.ID] = s
	}
	return &Store{Base: base, shards: imap}
}

func openWithRetry(dsn string, attempts int, sleep time.Duration) (*gorm.DB, error) {
	var last error
	for i := 1; i <= attempts; i++ {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			if s, e := db.DB(); e == nil && s != nil {
				if perr := pingWithTimeout(s, 2*time.Second); perr == nil {
					return db, nil
				} else {
					last = perr
				}
			} else {
				last = e
			}
		} else {
			last = err
		}
		time.Sleep(sleep)
		if sleep < 8*time.Second {
			sleep *= 2
		}
	}
	return nil, last
}

func pingWithTimeout(sqlDB *sql.DB, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- sqlDB.Ping() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("db ping timeout after %s", timeout)
	}
}

// Redact a few DSN kvs in logs (best-effort)
var reKV = regexp.MustCompile(`\b(host|port|dbname)=\S+`)

func RedactDSN(dsn string) string {
	parts := reKV.FindAllString(dsn, -1)
	if len(parts) == 0 {
		if len(dsn) > 48 {
			return dsn[:48] + "…"
		}
		return dsn
	}
	return fmt.Sprintf("%s", parts)
}

internal/shared/httpx/httpx.go
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"users-service/internal/shared/jwt"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) error

func Wrap(fn HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			code := http.StatusBadRequest
			if errors.Is(err, ErrUnauthorized) {
				code = http.StatusUnauthorized
			}
			WriteJSON(w, map[string]any{"error": err.Error()}, code)
		}
	})
}

func Decode[T any](r *http.Request) (T, error) {
	var t T
	err := json.NewDecoder(r.Body).Decode(&t)
	return t, err
}

func WriteJSON(w http.ResponseWriter, v any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

var (
	// Use stable string keys to avoid mismatches if multiple copies of the package are linked.
	ctxUserIDKey  = "httpx.user_id"
	ctxShardIDKey = "httpx.shard_id"

	ErrUnauthorized = errors.New("unauthorized")
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			WriteJSON(w, map[string]any{"error": "unauthorized", "reason": "missing bearer"}, http.StatusUnauthorized)
			return
		}
		tok := strings.TrimSpace(h[7:])
		uid, sh, err := jwt.Parse(tok)
		if err != nil || uid == "" {
			WriteJSON(w, map[string]any{"error": "unauthorized", "reason": "bad token"}, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserIDKey, uid)
		ctx = context.WithValue(ctx, ctxShardIDKey, sh)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromCtx(r *http.Request) (string, int, error) {
	uid, _ := r.Context().Value(ctxUserIDKey).(string)
	sh, _ := r.Context().Value(ctxShardIDKey).(int)
	if uid == "" {
		return "", 0, ErrUnauthorized
	}
	return uid, sh, nil
}

func QueryInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

internal/shared/jwt/jwt.go
package jwt

import (
	"errors"
	"os"
	"time"

	jw "github.com/golang-jwt/jwt/v5"
)

func secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("replace-this-with-a-strong-secret")
}

func Make(userID string, shardID int) (string, error) {
	claims := jw.MapClaims{
		"sub": userID,
		"sh":  shardID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	return jw.NewWithClaims(jw.SigningMethodHS256, claims).SignedString(secret())
}

func Parse(tok string) (string, int, error) {
	t, err := jw.Parse(tok, func(t *jw.Token) (any, error) { return secret(), nil })
	if err != nil || !t.Valid {
		return "", 0, errors.New("invalid token")
	}
	mc, ok := t.Claims.(jw.MapClaims)
	if !ok {
		return "", 0, errors.New("bad claims")
	}
	uid, _ := mc["sub"].(string)
	shf, ok := mc["sh"].(float64)
	if !ok {
		return "", 0, errors.New("missing shard")
	}
	return uid, int(shf), nil
}

internal/shared/shard/shard.go
package shard

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"
)

func Pick(key string, n int) int {
	h := sha256.Sum256([]byte(key))
	v := binary.BigEndian.Uint32(h[:4]) ^ binary.BigEndian.Uint32(h[4:8])
	return int(uint32(v) % uint32(n))
}
func Extract(userID string) (int, bool) {
	i := strings.IndexByte(userID, '-')
	if i <= 0 {
		return 0, false
	}
	n, err := strconv.Atoi(userID[:i])
	if err != nil {
		return 0, false
	}
	return n, true
}

internal/shared/validate/validate.go
package validate

import "github.com/go-playground/validator/v10"

var v = validator.New()

func Struct(s any) error { return v.Struct(s) }

internal/social/handler.go
package social

import (
	"net/http"

	"users-service/internal/shared/httpx"
	"users-service/internal/shared/validate"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Follow(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	target := r.PathValue("target_id")
	if target == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Follow(uid, target); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) Unfollow(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	target := r.PathValue("target_id")
	if target == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Unfollow(uid, target); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) ListFollowing(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.ListFollowing(uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) Befriend(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	friend := r.PathValue("friend_id")
	if friend == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Befriend(uid, friend); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) Unfriend(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	friend := r.PathValue("friend_id")
	if friend == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Unfriend(uid, friend); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.ListFriends(uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

type relReq struct {
	RelatedID string `json:"related_id" validate:"required"`
	Type      int    `json:"type" validate:"required"`
}

func (h *Handler) CreateRelationship(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[relReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	if err := h.svc.CreateRelationship(uid, in.RelatedID, in.Type); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) DeleteRelationship(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[relReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	if err := h.svc.DeleteRelationship(uid, in.RelatedID, in.Type); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) ListRelationships(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	typ := httpx.QueryInt(r, "type", 0) // 0 = any
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.ListRelationships(uid, typ, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{
		"items":  items,
		"type":   typ,
		"limit":  limit,
		"offset": offset,
	}, http.StatusOK)
	return nil
}

internal/social/reltypes.go
package social

const (
	RelTypeFollow = 1
	RelTypeFriend = 2
	RelTypeBlock  = 3
)

internal/social/repository.go
package social

import (
	"errors"

	"users-service/internal/shared/db"
	"users-service/internal/shared/shard"
	"users-service/internal/user"
)

type Repository interface {
	Follow(uid, target string) error
	Unfollow(uid, target string) error
	ListFollowing(uid string, limit, offset int) ([]string, error)

	Befriend(a, b string) error
	Unfriend(a, b string) error
	ListFriends(uid string, limit, offset int) ([]string, error)

	CreateRelationship(uid, related string, typ int) error
	DeleteRelationship(uid, related string, typ int) error
	ListRelationships(uid string, typ, limit, offset int) ([]string, error)
}

type repo struct {
	store *db.Store
	users user.Repository
}

func NewRepository(s *db.Store, ur user.Repository) Repository { return &repo{store: s, users: ur} }

func (r *repo) ensureUser(uid string) error {
	_, err := r.users.GetByUserID(uid)
	return err
}

func (r *repo) Follow(uid, target string) error {
	if uid == target {
		return errors.New("cannot follow self")
	}
	if err := r.ensureUser(target); err != nil {
		return errors.New("target not found")
	}
	sh, _ := shard.Extract(uid)
	return r.store.Write(sh).FirstOrCreate(&Follow{UserID: uid, TargetID: target}).Error
}
func (r *repo) Unfollow(uid, target string) error {
	sh, _ := shard.Extract(uid)
	return r.store.Write(sh).Delete(&Follow{}, "user_id=? AND target_id=?", uid, target).Error
}
func (r *repo) ListFollowing(uid string, limit, offset int) ([]string, error) {
	sh, _ := shard.Extract(uid)
	type Row struct{ TargetID string }
	var rows []Row
	if err := r.store.Use(sh).Model(&Follow{}).
		Where("user_id = ?", uid).Order("created_at DESC").
		Limit(limit).Offset(offset).Select("target_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].TargetID
	}
	return out, nil
}

func (r *repo) Befriend(a, b string) error {
	if a == b {
		return errors.New("cannot friend self")
	}
	if err := r.ensureUser(b); err != nil {
		return errors.New("target not found")
	}
	sha, _ := shard.Extract(a)
	shb, _ := shard.Extract(b)
	if err := r.store.Write(sha).FirstOrCreate(&Friend{UserID: a, FriendID: b}).Error; err != nil {
		return err
	}
	return r.store.Write(shb).FirstOrCreate(&Friend{UserID: b, FriendID: a}).Error
}
func (r *repo) Unfriend(a, b string) error {
	sha, _ := shard.Extract(a)
	shb, _ := shard.Extract(b)
	if err := r.store.Write(sha).Delete(&Friend{}, "user_id=? AND friend_id=?", a, b).Error; err != nil {
		return err
	}
	return r.store.Write(shb).Delete(&Friend{}, "user_id=? AND friend_id=?", b, a).Error
}
func (r *repo) ListFriends(uid string, limit, offset int) ([]string, error) {
	sh, _ := shard.Extract(uid)
	type Row struct{ FriendID string }
	var rows []Row
	if err := r.store.Use(sh).Model(&Friend{}).
		Where("user_id = ?", uid).Order("created_at DESC").
		Limit(limit).Offset(offset).Select("friend_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].FriendID
	}
	return out, nil
}

func (r *repo) CreateRelationship(uid, related string, typ int) error {
	if uid == related {
		return errors.New("cannot relate to self")
	}
	if err := r.ensureUser(related); err != nil {
		return errors.New("target not found")
	}
	sh, _ := shard.Extract(uid)
	rel := &Relationship{UserID: uid, RelatedID: related, Type: typ}
	return r.store.Write(sh).FirstOrCreate(rel).Error
}

func (r *repo) DeleteRelationship(uid, related string, typ int) error {
	sh, _ := shard.Extract(uid)
	return r.store.Write(sh).Delete(&Relationship{}, "user_id=? AND related_id=? AND type=?", uid, related, typ).Error
}

func (r *repo) ListRelationships(uid string, typ, limit, offset int) ([]string, error) {
	sh, _ := shard.Extract(uid)
	type Row struct{ RelatedID string }
	var rows []Row
	dbq := r.store.Use(sh).Model(&Relationship{}).Where("user_id = ?", uid)
	if typ != 0 {
		dbq = dbq.Where("type = ?", typ)
	}
	if err := dbq.Order("created_at DESC").
		Limit(limit).Offset(offset).
		Select("related_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].RelatedID
	}
	return out, nil
}

internal/social/service.go
package social

type Service interface {
	Follow(uid, target string) error
	Unfollow(uid, target string) error
	ListFollowing(uid string, limit, offset int) ([]string, error)
	Befriend(a, b string) error
	Unfriend(a, b string) error
	ListFriends(uid string, limit, offset int) ([]string, error)
	CreateRelationship(uid, related string, typ int) error
	DeleteRelationship(uid, related string, typ int) error
	ListRelationships(uid string, typ, limit, offset int) ([]string, error)
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Follow(uid, target string) error   { return s.repo.Follow(uid, target) }
func (s *service) Unfollow(uid, target string) error { return s.repo.Unfollow(uid, target) }
func (s *service) ListFollowing(uid string, limit, offset int) ([]string, error) {
	return s.repo.ListFollowing(uid, limit, offset)
}
func (s *service) Befriend(a, b string) error { return s.repo.Befriend(a, b) }
func (s *service) Unfriend(a, b string) error { return s.repo.Unfriend(a, b) }
func (s *service) ListFriends(uid string, limit, offset int) ([]string, error) {
	return s.repo.ListFriends(uid, limit, offset)
}
func (s *service) CreateRelationship(uid, related string, typ int) error {
	return s.repo.CreateRelationship(uid, related, typ)
}
func (s *service) DeleteRelationship(uid, related string, typ int) error {
	return s.repo.DeleteRelationship(uid, related, typ)
}
func (s *service) ListRelationships(uid string, typ, limit, offset int) ([]string, error) {
	return s.repo.ListRelationships(uid, typ, limit, offset)
}

internal/social/social.go
package social

import "time"

type Follow struct {
	UserID    string `gorm:"primaryKey;size:64"`
	TargetID  string `gorm:"primaryKey;size:64"`
	CreatedAt time.Time
}
type Friend struct {
	UserID    string `gorm:"primaryKey;size:64"`
	FriendID  string `gorm:"primaryKey;size:64"`
	CreatedAt time.Time
}
type Relationship struct {
	UserID    string `gorm:"primaryKey;size:64"`
	RelatedID string `gorm:"primaryKey;size:64"`
	Type      int    `gorm:"primaryKey"`
	CreatedAt time.Time
}

internal/user/handler.go
package user

import (
	"net/http"

	"users-service/internal/shared/httpx"
	"users-service/internal/shared/jwt"
	"users-service/internal/shared/validate"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) error {
	body, err := httpx.Decode[RegisterReq](r)
	if err != nil {
		return err
	}
	if err = validate.Struct(body); err != nil {
		return err
	}
	u, err := h.svc.Register(body.Email, body.Password, body.Name)
	if err != nil {
		return err
	}
	token, _ := jwt.Make(u.UserID, u.ShardID)
	httpx.WriteJSON(w, map[string]any{
		"user_id": u.UserID, "name": u.Name, "email": u.Email, "access_token": token,
	}, http.StatusCreated)
	return nil
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	body, err := httpx.Decode[LoginReq](r)
	if err != nil {
		return err
	}
	if err = validate.Struct(body); err != nil {
		return err
	}
	u, err := h.svc.Login(body.Email, body.Password)
	if err != nil {
		return err
	}
	token, _ := jwt.Make(u.UserID, u.ShardID)
	httpx.WriteJSON(w, map[string]any{
		"message": "login successful", "user_id": u.UserID, "name": u.Name, "email": u.Email, "access_token": token,
	}, http.StatusOK)
	return nil
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	uid := r.PathValue("user_id")
	u, err := h.svc.GetByUserID(uid)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, u, http.StatusOK)
	return nil
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) error {
	_, shardID, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	users, err := h.svc.ListMine(shardID, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"shard_id": shardID, "limit": limit, "offset": offset, "items": users}, http.StatusOK)
	return nil
}

internal/user/repository.go
package user

import (
	"errors"
	"users-service/internal/shared/db"
	"users-service/internal/shared/shard"
)

type Repository interface {
	Create(u *User) (*User, error)
	GetByEmail(email string, shardID int) (*User, error)
	GetByUserID(uid string) (*User, error)
	ListByShard(shardID, limit, offset int) ([]User, error)
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(u *User) (*User, error) {
	if err := r.store.Write(u.ShardID).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}
func (r *repo) GetByEmail(email string, shardID int) (*User, error) {
	var u User
	err := r.store.Use(shardID).Where("email = ?", email).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}
func (r *repo) GetByUserID(uid string) (*User, error) {
	sh, ok := shard.Extract(uid)
	if !ok {
		return nil, errors.New("bad user_id")
	}
	var u User
	if err := r.store.Use(sh).Where("user_id = ?", uid).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
func (r *repo) ListByShard(shardID, limit, offset int) ([]User, error) {
	var out []User
	err := r.store.Use(shardID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error
	return out, err
}

internal/user/service.go
package user

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"

	"users-service/internal/shared/shard"

	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(email, password, name string) (*User, error)
	Login(email, password string) (*User, error)
	GetByUserID(uid string) (*User, error)
	ListMine(shardID, limit, offset int) ([]User, error)
}
type service struct {
	repo      Repository
	numShards int
}

func NewService(r Repository) Service {
	n := 1
	if s := os.Getenv("NUM_SHARDS"); s != "" {
		if v, e := strconv.Atoi(s); e == nil && v > 0 {
			n = v
		}
	}
	return &service{repo: r, numShards: n}
}

func (s *service) Register(email, password, name string) (*User, error) {
	sh := shard.Pick(email, s.numShards)
	if exist, _ := s.repo.GetByEmail(email, sh); exist != nil {
		return nil, errors.New("user exists")
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	uid := fmt.Sprintf("%d-%x", sh, binary.BigEndian.Uint64(b[:]))
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("hash fail")
	}
	return s.repo.Create(&User{
		UserID: uid, ShardID: sh, Email: email, PassHash: string(hash), Name: name,
	})
}
func (s *service) Login(email, password string) (*User, error) {
	sh := shard.Pick(email, s.numShards)
	u, err := s.repo.GetByEmail(email, sh)
	if err != nil {
		return nil, errors.New("wrong credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PassHash), []byte(password)) != nil {
		return nil, errors.New("wrong credentials")
	}
	return u, nil
}
func (s *service) GetByUserID(uid string) (*User, error) { return s.repo.GetByUserID(uid) }
func (s *service) ListMine(shardID, limit, offset int) ([]User, error) {
	return s.repo.ListByShard(shardID, limit, offset)
}

internal/user/user.go
package user

import "time"

type User struct {
	UserID    string    `gorm:"uniqueIndex;size:64" json:"user_id"`
	ShardID   int       `gorm:"index" json:"shard_id"`
	ID        uint      `gorm:"primaryKey" json:"-"`
	Email     string    `gorm:"uniqueIndex;size:120" json:"email"`
	PassHash  string    `gorm:"size:255" json:"-"`
	Name      string    `gorm:"size:100" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RegisterReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required"`
}
type LoginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

