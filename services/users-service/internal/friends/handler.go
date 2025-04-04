package friends

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
	router.HandleFunc("/users/friends/create", h.CreateFriend())
}

func (h *Handler) CreateFriend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			UserID   int `json:"user_id"`
			FriendID int `json:"friend_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := h.Service.CreateFriend(body.UserID, body.FriendID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]string{"status": "friend added"}, http.StatusOK)
	}
}
