package comments

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

// registers /posts/comments/ route
func NewHandler(router *http.ServeMux, deps HandlerDeps) {
	h := &Handler{
		config:  deps.Config,
		service: deps.Service,
	}

	router.HandleFunc("/posts/comments/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if err := h.createComment(w, r); err != nil {
				res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			}
		case http.MethodGet:
			if err := h.getAllComments(w, r); err != nil {
				res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			}
		default:
			res.Json(w, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		}
	})
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) error {
	// Path format: /posts/comments/{postID}
	idStr := strings.TrimPrefix(r.URL.Path, "/posts/comments/")
	postID64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		res.Json(w, map[string]string{"error": "invalid post id"}, http.StatusBadRequest)
		return err
	}

	var payload struct {
		UserID uint   `json:"user_id"`
		Name   string `json:"name"`
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		res.Json(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
		return err
	}

	if err := h.service.CreateComment(payload.UserID, uint(postID64), payload.Name, payload.Text); err != nil {
		return err
	}
	res.Json(w, map[string]string{"message": "comment created"}, http.StatusCreated)
	return nil
}

func (h *Handler) getAllComments(w http.ResponseWriter, r *http.Request) error {
	idStr := strings.TrimPrefix(r.URL.Path, "/posts/comments/")
	postID64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		res.Json(w, map[string]string{"error": "invalid post id"}, http.StatusBadRequest)
		return err
	}

	comments, err := h.service.GetComments(uint(postID64))
	if err != nil {
		return err
	}
	res.Json(w, comments, http.StatusOK)
	return nil
}
