package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	jw "github.com/golang-jwt/jwt/v5"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) error

type APIError struct {
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
	Status int    `json:"status"`
}

type ctxKey string

const userKey ctxKey = "user_id"

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

func secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("replace-this-with-a-strong-secret")
}

func parseJWT(tok string) (string, error) {
	t, err := jw.Parse(tok, func(t *jw.Token) (any, error) { return secret(), nil })
	if err != nil || !t.Valid {
		return "", errors.New("invalid token")
	}
	mc, ok := t.Claims.(jw.MapClaims)
	if !ok {
		return "", errors.New("bad claims")
	}
	uid, _ := mc["sub"].(string)
	if uid == "" {
		return "", errors.New("missing sub")
	}
	if exp, ok := mc["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return "", errors.New("token expired")
	}
	return uid, nil
}

func Wrap(fn HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			WriteError(w, http.StatusBadRequest, err, "")
		}
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	secret := os.Getenv("JWT_SECRET")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret == "" {
			next.ServeHTTP(w, r)
			return
		}
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "missing_bearer")
			return
		}
		token := strings.TrimSpace(h[7:])
		uid, err := parseJWT(token)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid_token")
			return
		}
		ctx := context.WithValue(r.Context(), userKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromCtx(r *http.Request) (string, error) {
	uid, _ := r.Context().Value(userKey).(string)
	if uid == "" {
		return "", ErrUnauthorized
	}
	return uid, nil
}

func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func NowUTC() time.Time { return time.Now().UTC() }
