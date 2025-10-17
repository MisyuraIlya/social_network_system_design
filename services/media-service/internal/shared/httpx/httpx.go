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

func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}
