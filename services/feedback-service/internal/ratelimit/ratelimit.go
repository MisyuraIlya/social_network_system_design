package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"feedback-gateway/internal/shared/httpx"

	"github.com/redis/go-redis/v9"
)

type Limiter struct{ R *redis.Client }

func New(r *redis.Client) *Limiter { return &Limiter{R: r} }

func (l *Limiter) AllowSliding(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, error) {
	k := "rl:" + key
	pipe := l.R.TxPipeline()
	incr := pipe.Incr(ctx, k)
	pipe.Expire(ctx, k, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}
	n := incr.Val()
	return n <= limit, n, nil
}

func (l *Limiter) LimitHTTP(limit int64, window time.Duration, keyFn func(*http.Request) (string, error), next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, err := keyFn(r)
		if err != nil || key == "" {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrUnauthorized, "missing_user")
			return
		}
		ok, n, e := l.AllowSliding(r.Context(), key, limit, window)
		if e != nil {
			httpx.WriteError(w, http.StatusTooManyRequests, fmt.Errorf("rate limiter error"), "rate_limiter_error")
			return
		}
		if !ok {
			httpx.WriteError(w, http.StatusTooManyRequests,
				fmt.Errorf("rate limit exceeded (count=%d, limit=%d)", n, limit),
				"rate_limited")
			return
		}
		next.ServeHTTP(w, r)
	})
}
