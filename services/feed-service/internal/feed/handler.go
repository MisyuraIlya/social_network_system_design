package feed

import (
	"context"
	"encoding/json"
	"feed-service/pkg/res"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Handler represents our feed HTTP handler.
type Handler struct {
	service Service
}

// NewHandler initializes the handler and registers endpoints.
func NewHandler(router *http.ServeMux, service Service) {
	handler := &Handler{service: service}
	// Register routes with method prefixes.
	router.HandleFunc("GET /feed", handler.getFeed())
	router.HandleFunc("POST /feed", handler.createFeed())
	router.HandleFunc("GET /health", handler.healthCheck())
}

// healthCheck returns a simple health check endpoint.
func (h *Handler) healthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res.Json(w, map[string]string{"status": "OK"}, http.StatusOK)
	}
}

// createFeed handles POST /feed requests.
func (h *Handler) createFeed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			res.Json(w, map[string]string{"error": "Content-Type must be application/json"}, http.StatusBadRequest)
			return
		}

		var req CreateFeedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("decode error: %v", err)
			res.Json(w, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		resp, err := h.service.CreateFeedItem(ctx, req)
		if err != nil {
			log.Printf("Error creating feed item: %v", err)
			res.Json(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}

		res.Json(w, resp, http.StatusOK)
	}
}

// getFeed handles GET /feed requests.
func (h *Handler) getFeed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			res.Json(w, map[string]string{"error": "missing user_id"}, http.StatusBadRequest)
			return
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page == 0 {
			page = 1
		}

		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		if pageSize == 0 {
			pageSize = 10
		}

		ctx := r.Context()
		items, err := h.service.GetFeed(ctx, userID, page, pageSize)
		if err != nil {
			log.Printf("Error getting feed: %v", err)
			res.Json(w, map[string]string{"error": "failed to get feed"}, http.StatusInternalServerError)
			return
		}

		res.Json(w, items, http.StatusOK)
	}
}
