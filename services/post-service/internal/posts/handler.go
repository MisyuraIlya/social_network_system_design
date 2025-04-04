package posts

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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

func NewHandler(router *http.ServeMux, deps HandlerDeps) {
	h := &Handler{
		config:  deps.Config,
		service: deps.Service,
	}

	router.HandleFunc("/posts/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/posts")
		if path == "" || path == "/" {
			// Handle /posts
			switch r.Method {
			case http.MethodPost:
				if err := h.create(w, r); err != nil {
					res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
				}
			case http.MethodGet:
				if err := h.getAll(w, r); err != nil {
					res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
				}
			default:
				res.Json(w, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
			}
			return
		}

		// Handle /posts/{id}
		idStr := strings.TrimPrefix(path, "/")
		id64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			res.Json(w, map[string]string{"error": "invalid post id"}, http.StatusBadRequest)
			return
		}
		id := uint(id64)

		switch r.Method {
		case http.MethodGet:
			if err := h.getByID(w, r, id); err != nil {
				res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			}
		case http.MethodPut:
			if err := h.update(w, r, id); err != nil {
				res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			}
		case http.MethodDelete:
			if err := h.delete(w, r, id); err != nil {
				res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			}
		default:
			res.Json(w, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		}
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var payload struct {
		UserID      uint   `json:"user_id"`
		Description string `json:"description"`
		Media       string `json:"media"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		res.Json(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
		return err
	}

	if err := h.service.Create(payload.UserID, payload.Description, payload.Media); err != nil {
		return err
	}

	res.Json(w, map[string]string{"message": "post created"}, http.StatusCreated)
	return nil
}

func (h *Handler) getAll(w http.ResponseWriter, r *http.Request) error {
	posts, err := h.service.GetAll()
	if err != nil {
		return err
	}
	res.Json(w, posts, http.StatusOK)
	return nil
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request, id uint) error {
	post, err := h.service.GetByID(id)
	if err != nil {
		return err
	}
	if post == nil {
		res.Json(w, map[string]string{"error": "post not found"}, http.StatusNotFound)
		return nil
	}
	res.Json(w, post, http.StatusOK)
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id uint) error {
	var payload struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		res.Json(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
		return err
	}

	if err := h.service.Update(id, payload.Description); err != nil {
		return err
	}
	res.Json(w, map[string]string{"message": "post updated"}, http.StatusOK)
	return nil
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, id uint) error {
	if err := h.service.Delete(id); err != nil {
		return err
	}
	res.Json(w, map[string]string{"message": "post deleted"}, http.StatusOK)
	return nil
}
