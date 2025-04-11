package posts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"post-service/configs"
	"post-service/pkg/res"
)

type Handler struct {
	config  *configs.Config
	service Service
}

func NewHandler(router *http.ServeMux, config *configs.Config, service Service) {
	handler := &Handler{
		config:  config,
		service: service,
	}

	fmt.Println("Registering posts routes...")
	router.HandleFunc("POST /posts", handler.create())
	router.HandleFunc("GET /posts", handler.getAll())
	router.HandleFunc("GET /posts/{id}", handler.getByID())
	router.HandleFunc("PUT /posts/", handler.update())
	router.HandleFunc("DELETE /posts/", handler.delete())
}

// create handles POST /posts requests.
func (h *Handler) create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			UserID      uint   `json:"user_id"`
			Description string `json:"description"`
			Media       string `json:"media"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
			return
		}

		fmt.Printf("Received POST payload: %+v\n", payload)

		if err := h.service.Create(payload.UserID, payload.Description, payload.Media); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}

		res.Json(w, map[string]string{"message": "post created"}, http.StatusCreated)
	}
}

// getAll handles GET /posts requests.
func (h *Handler) getAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Fetching all posts...")
		posts, err := h.service.GetAll()
		if err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}

		fmt.Printf("Fetched %d posts from database\n", len(posts))
		res.Json(w, posts, http.StatusOK)
	}
}

// getByID handles GET /posts/{id} requests.
func (h *Handler) getByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the id from the URL (assumes URL is "/posts/{id}")
		idStr := strings.TrimPrefix(r.URL.Path, "/posts/")
		id64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			res.Json(w, map[string]string{"error": "invalid post id"}, http.StatusBadRequest)
			return
		}
		id := uint(id64)

		post, err := h.service.GetByID(id)
		if err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}
		if post == nil {
			res.Json(w, map[string]string{"error": "post not found"}, http.StatusNotFound)
			return
		}
		res.Json(w, post, http.StatusOK)
	}
}

// update handles PUT /posts/{id} requests.
func (h *Handler) update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the id from the URL (assumes URL is "/posts/{id}")
		idStr := strings.TrimPrefix(r.URL.Path, "/posts/")
		id64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			res.Json(w, map[string]string{"error": "invalid post id"}, http.StatusBadRequest)
			return
		}
		id := uint(id64)

		var payload struct {
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
			return
		}

		if err := h.service.Update(id, payload.Description); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]string{"message": "post updated"}, http.StatusOK)
	}
}

// delete handles DELETE /posts/{id} requests.
func (h *Handler) delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the id from the URL (assumes URL is "/posts/{id}")
		idStr := strings.TrimPrefix(r.URL.Path, "/posts/")
		id64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			res.Json(w, map[string]string{"error": "invalid post id"}, http.StatusBadRequest)
			return
		}
		id := uint(id64)

		if err := h.service.Delete(id); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]string{"message": "post deleted"}, http.StatusOK)
	}
}
