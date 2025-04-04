package userdata

import (
	"net/http"
	"strconv"
	"strings"
	"users-service/configs"
	"users-service/pkg/req"
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
	router.HandleFunc("/users/userdata/get", h.GetUserData())
	router.HandleFunc("/users/userdata/update", h.UpdateUserData())
	router.HandleFunc("/users/", h.GetUserByID())
}

func (h *Handler) GetUserData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userIDStr := r.URL.Query().Get("user_id")
		if userIDStr == "" {
			http.Error(w, "missing user_id", http.StatusBadRequest)
			return
		}
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			http.Error(w, "invalid user_id", http.StatusBadRequest)
			return
		}
		data, err := h.Service.GetUserData(userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		res.Json(w, data, http.StatusOK)
	}
}

func (h *Handler) UpdateUserData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := req.HandleBody[UserData](&w, r)
		if err != nil {
			return
		}

		if err := h.Service.UpdateUserData(body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Json(w, body, http.StatusOK)
	}
}

func (h *Handler) GetUserByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// expected path: /users/123
		id := strings.TrimPrefix(r.URL.Path, "/users/")
		userID, err := strconv.Atoi(id)
		if err != nil {
			http.Error(w, "invalid user_id", http.StatusBadRequest)
			return
		}

		data, err := h.Service.GetUserData(userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		res.Json(w, data, http.StatusOK)
	}
}
