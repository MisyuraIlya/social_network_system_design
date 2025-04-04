package gateway

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func ValidateToken(token, userServiceURL string) error {
	if token == "" {
		return errors.New("missing token")
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/auth/validate", userServiceURL), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("unauthorized")
	}

	return nil
}

func ExtractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("missing or malformed token")
	}
	return strings.TrimPrefix(authHeader, "Bearer "), nil
}
