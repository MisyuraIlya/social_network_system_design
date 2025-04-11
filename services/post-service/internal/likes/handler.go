package likes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"post-service/configs"
	"post-service/pkg/res"
)

// Service interface should declare the method used by the handler,
// for example LikePost.

type Handler struct {
	config  *configs.Config
	service Service
}

// NewHandler registers the likes routes. It follows a similar pattern
// to the comments package, where the configuration and service are passed
// directly as arguments.
func NewHandler(router *http.ServeMux, config *configs.Config, service Service) {
	handler := &Handler{
		config:  config,
		service: service,
	}

	fmt.Println("Registering likes routes...")
	router.HandleFunc("POST /posts/like/", handler.likePost())
}

// likePost handles POST /posts/like/{postID} requests.
// It extracts the post id from the URL, decodes the JSON payload, and
// invokes the service layer to add a like.
func (h *Handler) likePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Expected URL format: /posts/like/{postID}
		idStr := strings.TrimPrefix(r.URL.Path, "/posts/like/")
		postID64, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			res.Json(w, map[string]string{"error": "invalid post id"}, http.StatusBadRequest)
			return
		}

		var payload struct {
			UserID    uint `json:"user_id"`
			CommentID uint `json:"comment_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
			return
		}

		if err := h.service.LikePost(payload.UserID, uint(postID64), payload.CommentID); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}

		res.Json(w, map[string]string{"message": "like added"}, http.StatusOK)
	}
}
