package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"feed-service/internal/shared/jwt"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) error

type APIError struct {
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
	Status int    `json:"status"`
}

var (
	ctxUserIDKey    = "httpx.user_id"
	ErrUnauthorized = errors.New("unauthorized")
)

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

func Wrap(fn HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			code := http.StatusBadRequest
			if errors.Is(err, ErrUnauthorized) {
				code = http.StatusUnauthorized
			}
			WriteError(w, code, err, "")
		}
	})
}

func Decode[T any](r *http.Request) (T, error) {
	var t T
	err := json.NewDecoder(r.Body).Decode(&t)
	return t, err
}

func WriteBadRequest(w http.ResponseWriter, err error, reason string) {
	WriteError(w, http.StatusBadRequest, err, reason)
}

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
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "missing_bearer")
			return
		}
		uid, err := jwt.Parse(tok)
		if err != nil || uid == "" {
			WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid_token")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserIDKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromCtx(r *http.Request) (string, error) {
	uid, _ := r.Context().Value(ctxUserIDKey).(string)
	if uid == "" {
		return "", ErrUnauthorized
	}
	return uid, nil
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
