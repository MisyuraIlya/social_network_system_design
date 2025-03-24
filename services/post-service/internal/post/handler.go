package post

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type PostHandler struct {
	Service IPostService
}

func NewPostHandler(svc IPostService) *PostHandler {
	return &PostHandler{Service: svc}
}

func RegisterRoutes(mux *http.ServeMux, handler *PostHandler) {
	mux.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.ListPosts(w, r)
		case http.MethodPost:
			handler.Create(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/posts/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 2 {
			http.Error(w, "Post ID missing", http.StatusBadRequest)
			return
		}
		idStr := pathParts[1]
		handler.GetPost(w, r, idStr)
	})
}

func (h *PostHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := h.Service.ListPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(posts)
}

func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID  uint   `json:"userId"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	post, err := h.Service.CreatePost(body.UserID, body.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}

func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid post ID", http.StatusBadRequest)
		return
	}
	post, err := h.Service.GetPost(uint(id))
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(post)
}
