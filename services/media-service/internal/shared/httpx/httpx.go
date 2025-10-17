package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const userKey ctxKey = "uid"

type APIError struct {
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
	Status int    `json:"status"`
}

var ErrUnauthorized = errors.New("unauthorized")

func WriteJSON(w http.ResponseWriter, v any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, err error, reason string) {
	if err == nil {
		err = errors.New(http.StatusText(status))
	}
	WriteJSON(w, APIError{Error: err.Error(), Reason: reason, Status: status}, status)
}

func AuthMiddleware(next http.Handler) http.Handler {
	secret := os.Getenv("JWT_SECRET")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret == "" {
			// dev mode: attach dummy uid "0"
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, "0")))
			return
		}
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "missing_bearer")
			return
		}
		token := strings.TrimSpace(h[7:])
		parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) { return []byte(secret), nil })
		if err != nil || !parsed.Valid {
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid_token")
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
		return "", ErrUnauthorized
	}
	return v, nil
}

func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}
