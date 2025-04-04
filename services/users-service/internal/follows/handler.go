package follows

import (
	"encoding/json"
	"net/http"
	"users-service/configs"
	"users-service/pkg/res"
)

type HandlerDeps struct {
	*configs.Config
	Service Service
}

type Handler struct {
	*configs.Config
	Service Service
}

func NewHandler(router *http.ServeMux, deps HandlerDeps) {
	h := &Handler{
		Config:  deps.Config,
		Service: deps.Service,
	}
	router.HandleFunc("/users/follows/create", h.CreateFollow())
}

func (h *Handler) CreateFollow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			UserID     int `json:"user_id"`
			FollowedID int `json:"followed_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := h.Service.CreateFollow(body.UserID, body.FollowedID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]string{"status": "followed"}, http.StatusOK)
	}
}
