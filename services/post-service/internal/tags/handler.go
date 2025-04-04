package tags

import (
	"encoding/json"
	"net/http"

	"post-service/configs"
	"post-service/pkg/res"
)

type HandlerDeps struct {
	Config  *configs.Config
	Service Service
}

type Handler struct {
	config  *configs.Config
	service Service
}

// registers /tags
func NewHandler(router *http.ServeMux, deps HandlerDeps) {
	h := &Handler{
		config:  deps.Config,
		service: deps.Service,
	}

	router.HandleFunc("/posts/tags", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if err := h.createTag(w, r); err != nil {
				res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			}
		case http.MethodGet:
			if err := h.getAllTags(w, r); err != nil {
				res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			}
		default:
			res.Json(w, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		}
	})
}

func (h *Handler) createTag(w http.ResponseWriter, r *http.Request) error {
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		res.Json(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
		return err
	}
	if err := h.service.CreateTag(payload.Name); err != nil {
		return err
	}
	res.Json(w, map[string]string{"message": "tag created"}, http.StatusCreated)
	return nil
}

func (h *Handler) getAllTags(w http.ResponseWriter, r *http.Request) error {
	allTags, err := h.service.GetAllTags()
	if err != nil {
		return err
	}
	res.Json(w, allTags, http.StatusOK)
	return nil
}
