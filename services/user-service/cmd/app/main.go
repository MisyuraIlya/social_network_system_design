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
