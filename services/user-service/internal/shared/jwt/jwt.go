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
	return []byte("replace-this-with-a-strong-secret")
}

func Make(userID string, shardID int) (string, error) {
	claims := jw.MapClaims{
		"sub": userID,
		"sh":  shardID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	return jw.NewWithClaims(jw.SigningMethodHS256, claims).SignedString(secret())
}

func Parse(tok string) (string, int, error) {
	t, err := jw.Parse(tok, func(t *jw.Token) (any, error) { return secret(), nil })
	if err != nil || !t.Valid {
		return "", 0, errors.New("invalid token")
	}
	mc, ok := t.Claims.(jw.MapClaims)
	if !ok {
		return "", 0, errors.New("bad claims")
	}
	uid, _ := mc["sub"].(string)
	shf, ok := mc["sh"].(float64)
	if !ok {
		return "", 0, errors.New("missing shard")
	}
	return uid, int(shf), nil
}
