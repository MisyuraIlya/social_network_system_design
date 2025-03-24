package feed

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type FeedHandler struct {
	Service IFeedService
}

func NewFeedHandler(s IFeedService) *FeedHandler {
	return &FeedHandler{Service: s}
}

func RegisterRoutes(mux *http.ServeMux, h *FeedHandler) {
	mux.HandleFunc("/feed", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.AddToFeed(w, r)
		case http.MethodGet:
			h.GetUserFeed(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/feed/user/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 {
			http.Error(w, "Missing user ID", http.StatusBadRequest)
			return
		}
		userIDStr := parts[2]
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		h.HandleGetUserFeed(w, r, uint(userID))
	})
}

func (h *FeedHandler) AddToFeed(w http.ResponseWriter, r *http.Request) {
	var body AddToFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item, err := h.Service.AddToFeed(body.UserID, body.PostID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := AddToFeedResponse{
		ID:     item.ID,
		UserID: item.UserID,
		PostID: item.PostID,
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *FeedHandler) GetUserFeed(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("userId")
	if userIDStr == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid userId", http.StatusBadRequest)
		return
	}
	h.HandleGetUserFeed(w, r, uint(userID))
}

func (h *FeedHandler) HandleGetUserFeed(w http.ResponseWriter, r *http.Request, userID uint) {
	items, err := h.Service.GetFeedForUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := make([]FeedItemData, 0, len(items))
	for _, it := range items {
		data = append(data, FeedItemData{
			ID:        it.ID,
			UserID:    it.UserID,
			PostID:    it.PostID,
			CreatedAt: it.CreatedAt,
		})
	}
	resp := GetFeedResponse{Items: data}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
