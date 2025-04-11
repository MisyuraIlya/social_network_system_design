package tags

import (
	"encoding/json"
	"fmt"
	"net/http"

	"post-service/configs"
	"post-service/pkg/res"
)

type Handler struct {
	config  *configs.Config
	service Service
}

// NewHandler registers the tags routes. It creates a new Handler and
// maps the routes for POST and GET requests to /posts/tags.
func NewHandler(router *http.ServeMux, config *configs.Config, service Service) {
	handler := &Handler{
		config:  config,
		service: service,
	}

	fmt.Println("Registering tags routes...")
	router.HandleFunc("POST /posts/tags", handler.createTag())
	router.HandleFunc("GET /posts/tags", handler.getAllTags())
}

// createTag handles POST /posts/tags requests.
// It decodes the JSON payload to extract the tag name and
// calls the service to create the new tag.
func (h *Handler) createTag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
			return
		}

		if err := h.service.CreateTag(payload.Name); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}

		res.Json(w, map[string]string{"message": "tag created"}, http.StatusCreated)
	}
}

// getAllTags handles GET /posts/tags requests.
// It retrieves all tags via the service and returns them as JSON.
func (h *Handler) getAllTags() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allTags, err := h.service.GetAllTags()
		if err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}
		res.Json(w, allTags, http.StatusOK)
	}
}
