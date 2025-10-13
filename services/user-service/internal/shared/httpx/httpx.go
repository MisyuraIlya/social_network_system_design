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
