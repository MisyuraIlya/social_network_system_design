package req

import (
	"net/http"
	"users-service/pkg/res"
)

func HandleBody[T any](w *http.ResponseWriter, r *http.Request) (*T, error) {
	body, err := Decode[T](r.Body)
	if err != nil {
		res.Json(*w, map[string]any{"error": "invalid JSON"}, http.StatusBadRequest)
		return nil, err
	}
	if err = IsValid(body); err != nil {
		res.Json(*w, map[string]any{"error": err.Error()}, http.StatusUnprocessableEntity)
		return nil, err
	}
	return &body, nil
}
