package user

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"users-service/internal/auth"
)

type UserHandler struct {
	Service IUserService
}

func NewUserHandler(svc IUserService) *UserHandler { return &UserHandler{Service: svc} }

func RegisterRoutes(mux *http.ServeMux, h *UserHandler) {
	// POST /users  -> register
	// GET  /users  -> list MY shard (requires JWT)
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.Register(w, r)
		case http.MethodGet:
			h.ListMine(w, r) // <-- lists only the caller's shard via JWT
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	// POST /users/login -> login + returns JWT
	mux.HandleFunc("/users/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		h.Login(w, r)
	})

	// GET /users/{user_id} -> fetch by id (service derives shard from user_id prefix)
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 2 {
			http.Error(w, "User ID missing", http.StatusBadRequest)
			return
		}
		h.GetUser(w, r, parts[1])
	})

	// OPTIONAL admin/dev route to peek a specific shard
	mux.HandleFunc("/admin/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		h.ListAdminByShard(w, r)
	})
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	u, err := h.Service.Register(body.Email, body.Password, body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// issue a JWT on register as well (handy for immediate login)
	tok, _ := auth.MakeJWT(u.UserID, u.ShardID)

	w.Header().Set("X-Shard-ID", strconv.Itoa(u.ShardID)) // debug only
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"user_id":      u.UserID,
		"email":        u.Email,
		"name":         u.Name,
		"access_token": tok,
	})
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	u, err := h.Service.Login(body.Email, body.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tok, _ := auth.MakeJWT(u.UserID, u.ShardID)

	w.Header().Set("X-Shard-ID", strconv.Itoa(u.ShardID)) // debug only
	json.NewEncoder(w).Encode(map[string]any{
		"message":      "login successful",
		"user_id":      u.UserID,
		"name":         u.Name,
		"email":        u.Email,
		"access_token": tok,
	})
}

func (h *UserHandler) GetUser(w http.ResponseWriter, _ *http.Request, userID string) {
	usr, err := h.Service.GetByUserID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(usr)
}

// GET /users -> lists MY shard, requires Authorization: Bearer <JWT>
// Supports optional ?limit=&offset=
func (h *UserHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	_, shardID, err := auth.ParseAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	users, err := h.Service.ListShard(shardID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"shard_id": shardID,
		"limit":    limit,
		"offset":   offset,
		"items":    users,
	})
}

// OPTIONAL admin/dev: GET /admin/users?shard=0&limit=50&offset=0
func (h *UserHandler) ListAdminByShard(w http.ResponseWriter, r *http.Request) {
	shStr := r.URL.Query().Get("shard")
	if shStr == "" {
		http.Error(w, "shard param required", http.StatusBadRequest)
		return
	}
	shardID, err := strconv.Atoi(shStr)
	if err != nil {
		http.Error(w, "invalid shard", http.StatusBadRequest)
		return
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	users, err := h.Service.ListShard(shardID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"shard_id": shardID,
		"limit":    limit,
		"offset":   offset,
		"items":    users,
	})
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
