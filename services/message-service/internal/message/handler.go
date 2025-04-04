package message

import (
	"encoding/json"
	"fmt"
	"message-service/configs"
	"message-service/pkg/res"
	"net/http"
)

type HandlerDeps struct {
	*configs.Config
	Service   Service
	Cache     Cache
	Publisher Publisher
}

type Handler struct {
	*configs.Config
	Service   Service
	Cache     Cache
	Publisher Publisher
}

func NewHandler(router *http.ServeMux, deps HandlerDeps) {
	h := &Handler{
		Config:    deps.Config,
		Service:   deps.Service,
		Cache:     deps.Cache,
		Publisher: deps.Publisher,
	}
	router.HandleFunc("/messages/create", h.CreateMessage())
	router.HandleFunc("/messages/list", h.ListMessages())
}

func (h *Handler) CreateMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			UserID  uint   `json:"user_id"`
			ChatID  uint   `json:"chat_id"`
			Content string `json:"content"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		msg, err := h.Service.CreateMessage(body.UserID, body.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// ✅ Update Redis popular chat count
		cacheKey := fmt.Sprintf("popular:chat:%d", body.ChatID)
		err = h.Cache.SetPopularChat(cacheKey, "true")
		if err != nil {
			fmt.Println("Redis error:", err)
		}

		// ✅ Publish Kafka event for new message
		event := struct {
			MessageID uint   `json:"message_id"`
			UserID    uint   `json:"user_id"`
			ChatID    uint   `json:"chat_id"`
			Content   string `json:"content"`
		}{
			MessageID: msg.ID,
			UserID:    body.UserID,
			ChatID:    body.ChatID,
			Content:   body.Content,
		}

		eventBytes, _ := json.Marshal(event)
		err = h.Publisher.PublishNewMessage(eventBytes)
		if err != nil {
			fmt.Println("Kafka publish error:", err)
		}

		res.Json(w, msg, http.StatusCreated)
	}
}

func (h *Handler) ListMessages() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		msgs, err := h.Service.ListMessages()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, msgs, http.StatusOK)
	}
}
