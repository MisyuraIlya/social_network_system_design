package likes

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

// registers /posts/like/
func NewHandler(router *http.ServeMux, deps HandlerDeps) {
	h := &Handler{
		config:  deps.Config,
		service: deps.Service,
	}

	router.HandleFunc("/posts/like/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			res.Json(w, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
			return
		}
		if err := h.likePost(w, r); err != nil {
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		}
	})
}

func (h *Handler) likePost(w http.ResponseWriter, r *http.Request) error {
	// /posts/like/{postID}
	idStr := strings.TrimPrefix(r.URL.Path, "/posts/like/")
	postID64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		res.Json(w, map[string]string{"error": "invalid post id"}, http.StatusBadRequest)
		return err
	}

	var body struct {
		UserID    uint `json:"user_id"`
		CommentID uint `json:"comment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		res.Json(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
		return err
	}

	err = h.service.LikePost(body.UserID, uint(postID64), body.CommentID)
	if err != nil {
		return err
	}
	res.Json(w, map[string]string{"message": "like added"}, http.StatusOK)
	return nil
}
