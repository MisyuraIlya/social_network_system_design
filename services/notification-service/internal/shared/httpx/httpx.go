package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"notification-service/internal/shared/jwt"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) error

func Wrap(fn HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusBadRequest)
		}
	})
}

func WriteJSON(w http.ResponseWriter, v any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type ctxKey string

const userKey ctxKey = "user_id"

var ErrNoUser = errors.New("no user in context")

func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := BearerToken(r)
		if tok == "" {
			WriteJSON(w, map[string]string{"error": "missing token"}, http.StatusUnauthorized)
			return
		}
		uid, err := jwt.Parse(tok)
		if err != nil || uid == "" {
			WriteJSON(w, map[string]string{"error": "invalid token"}, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromCtx(r *http.Request) (string, error) {
	uid, _ := r.Context().Value(userKey).(string)
	if uid == "" {
		return "", ErrNoUser
	}
	return uid, nil
}
