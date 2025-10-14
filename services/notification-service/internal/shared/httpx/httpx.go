package httpx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
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

func AuthMiddleware(next http.Handler) http.Handler {
	secret := os.Getenv("JWT_SECRET")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret == "" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			WriteJSON(w, map[string]string{"error": "missing token"}, http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		uid, err := verifyToken(token, secret)
		if err != nil {
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
		return "", errors.New("no user in context")
	}
	return uid, nil
}

func verifyToken(token, secret string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("bad token")
	}
	raw, sig := parts[0], parts[1]
	sum := hmac.New(sha256.New, []byte(secret))
	sum.Write([]byte(raw))
	expected := base64.RawURLEncoding.EncodeToString(sum.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", errors.New("bad sig")
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func MintToken(userID, secret string) string {
	raw := base64.RawURLEncoding.EncodeToString([]byte(userID))
	sum := hmac.New(sha256.New, []byte(secret))
	sum.Write([]byte(raw))
	sig := base64.RawURLEncoding.EncodeToString(sum.Sum(nil))
	return raw + "." + sig
}

func NowUTC() time.Time { return time.Now().UTC() }
