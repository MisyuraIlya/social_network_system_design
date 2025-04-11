package comments

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"post-service/configs"
	"post-service/pkg/res"
)

// Service interface should declare methods used by the handler,
// for instance CreateComment and GetComments.

type Handler struct {
	config  *configs.Config
	service Service
}

// NewHandler registers the comments routes. It follows a similar
// pattern to the posts package, with separate routes for POST and GET.
func NewHandler(router *http.ServeMux, config *configs.Config, service Service) {
	handler := &Handler{
		config:  config,
		service: service,
	}

	fmt.Println("Registering comments routes...")
	// These patterns assume a router that supports custom patterns.
	// If you’re using the default net/http ServeMux, note that
	// the registered string is a prefix match.
	router.HandleFunc("POST /posts/comments/", handler.createComment())
	router.HandleFunc("GET /posts/comments/", handler.getAllComments())
}

// createComment handles POST /posts/comments/{postID} requests.
// It extracts the post ID from the URL, decodes the JSON payload, and
// invokes the service layer to create a new comment.
func (h *Handler) createComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// URL format should be /posts/comments/{postID}
		idStr := strings.TrimPrefix(r.URL.Path, "/posts/comments/")
		postID64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			res.Json(w, map[string]string{"error": "invalid post id"}, http.StatusBadRequest)
			return
		}

		var payload struct {
			UserID uint   `json:"user_id"`
			Name   string `json:"name"`
			Text   string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
			return
		}

		if err := h.service.CreateComment(payload.UserID, uint(postID64), payload.Name, payload.Text); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}

		res.Json(w, map[string]string{"message": "comment created"}, http.StatusCreated)
	}
}

// getAllComments handles GET /posts/comments/{postID} requests.
// It extracts the post ID from the URL and retrieves all comments for that post.
func (h *Handler) getAllComments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/posts/comments/")
		postID64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			res.Json(w, map[string]string{"error": "invalid post id"}, http.StatusBadRequest)
			return
		}

		comments, err := h.service.GetComments(uint(postID64))
		if err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}

		res.Json(w, comments, http.StatusOK)
	}
}
