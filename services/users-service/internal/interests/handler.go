package interests

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
	router.HandleFunc("/users/interests/create", h.CreateInterest())
	router.HandleFunc("/users/interests/list", h.ListInterests())
	router.HandleFunc("/users/interests/add", h.AddUserInterest())
}

func (h *Handler) CreateInterest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		i, err := h.Service.CreateInterest(body.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, i, http.StatusOK)
	}
}

func (h *Handler) ListInterests() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		list, err := h.Service.ListInterests()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, list, http.StatusOK)
	}
}

func (h *Handler) AddUserInterest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			UserID     int `json:"user_id"`
			InterestID int `json:"interest_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := h.Service.AddUserInterest(body.UserID, body.InterestID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]string{"status": "interest added"}, http.StatusOK)
	}
}
