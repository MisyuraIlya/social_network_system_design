package user

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"users-service/pkg/req"
	"users-service/pkg/res"
)

type Handler struct {
	Service Service
}

func (h *Handler) HandleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getAllUsers(w, r)
	case http.MethodPost:
		h.createUser(w, r)
	default:
		res.Json(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) HandleUserByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/users/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		res.Json(w, "User ID is required", http.StatusBadRequest)
		return
	}
	idStr := parts[0]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		res.Json(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getUserByID(w, r, id)
	case http.MethodPut:
		h.updateUser(w, r, id)
	case http.MethodDelete:
		h.deleteUser(w, r, id)
	default:
		res.Json(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) getAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Service.GetAllUsers()
	if err != nil {
		res.Json(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res.Json(w, users, http.StatusOK)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	payload, err := req.HandleBody[UserCreatePayload](&w, r)
	if err != nil {
		return // error already handled in req.HandleBody
	}
	newUser, err := h.Service.CreateUser(*payload)
	if err != nil {
		res.Json(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res.Json(w, newUser, http.StatusCreated)
}

func (h *Handler) getUserByID(w http.ResponseWriter, r *http.Request, id int) {
	user, err := h.Service.GetUserByID(id)
	if err != nil {
		if errors.Is(err, errors.New("user not found")) {
			res.Json(w, err.Error(), http.StatusNotFound)
		} else {
			res.Json(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	res.Json(w, user, http.StatusOK)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request, id int) {
	payload, err := req.HandleBody[UserUpdatePayload](&w, r)
	if err != nil {
		return
	}
	updatedUser, err := h.Service.UpdateUser(id, *payload)
	if err != nil {
		if errors.Is(err, errors.New("user not found")) {
			res.Json(w, err.Error(), http.StatusNotFound)
		} else {
			res.Json(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	res.Json(w, updatedUser, http.StatusOK)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request, id int) {
	err := h.Service.DeleteUser(id)
	if err != nil {
		if errors.Is(err, errors.New("user not found")) {
			res.Json(w, err.Error(), http.StatusNotFound)
		} else {
			res.Json(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	res.Json(w, "User deleted successfully", http.StatusOK)
}
