package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("replace-this-with-a-strong-secret")

func MakeJWT(userID string, shardID int) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"sh":  shardID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(jwtKey)
}

func ParseAuthHeader(authz string) (userID string, shardID int, err error) {
	if authz == "" {
		return "", 0, errors.New("missing Authorization")
	}
	tokenStr := strings.TrimPrefix(authz, "Bearer ")
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return jwtKey, nil
	})
	if err != nil || !tok.Valid {
		return "", 0, errors.New("invalid token")
	}
	mc, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", 0, errors.New("bad claims")
	}
	uid, _ := mc["sub"].(string)
	// sh comes as float64 from JSON numbers
	shf, ok := mc["sh"].(float64)
	if !ok {
		return "", 0, errors.New("missing shard claim")
	}
	return uid, int(shf), nil
}
