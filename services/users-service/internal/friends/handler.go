package friends

import (
	"encoding/json"
	"net/http"
	"strconv"
	"users-service/configs"
	"users-service/pkg/middleware"
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
	// Using dynamic route parameter for the GET endpoint.
	router.Handle("POST /users/friends/create", middleware.IsAuthed(h.CreateFriend(), deps.Config))
	router.Handle("GET /users/{id}/friends", h.GetFriends())
}

func (h *Handler) CreateFriend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) GetFriends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the user id from the dynamic segment "{id}".
		idStr := r.PathValue("id")
		if idStr == "" {
			http.Error(w, "user id is required", http.StatusBadRequest)
			return
		}
		userID, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}
		friends, err := h.Service.GetFriends(userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, friends, http.StatusOK)
	}
}
