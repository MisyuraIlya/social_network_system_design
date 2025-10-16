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
	// dev fallback; replace in prod
	return []byte("replace-this-with-a-strong-secret")
}

// Parse validates HS256 JWT and returns the user id from the "sub" claim.
func Parse(tok string) (string, error) {
	t, err := jw.Parse(tok, func(t *jw.Token) (any, error) {
		return secret(), nil
	})
	if err != nil || !t.Valid {
		return "", errors.New("invalid token")
	}
	mc, ok := t.Claims.(jw.MapClaims)
	if !ok {
		return "", errors.New("bad claims")
	}
	uid, _ := mc["sub"].(string)
	if uid == "" {
		return "", errors.New("no subject")
	}
	// Optional but recommended: honor exp if present
	if exp, ok := mc["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return "", errors.New("token expired")
	}
	return uid, nil
}
