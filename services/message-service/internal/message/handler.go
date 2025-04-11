package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"

	"message-service/configs"
	"message-service/pkg/res"
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

// NewHandler wires up all HTTP routes.
func NewHandler(router *http.ServeMux, deps HandlerDeps) {
	h := &Handler{
		Config:    deps.Config,
		Service:   deps.Service,
		Cache:     deps.Cache,
		Publisher: deps.Publisher,
	}
	router.HandleFunc("/messages/create", h.CreateMessage())
	router.HandleFunc("/messages/list", h.ListMessages())
	router.HandleFunc("/messages/listByChat", h.ListMessagesByChat())
	router.HandleFunc("/messages/update", h.UpdateMessage())
	router.HandleFunc("/messages/delete", h.DeleteMessage())
	router.HandleFunc("/messages/uploadMedia", h.UploadMedia())
	router.HandleFunc("/messages/chats/popular", h.ListPopularChats())
}

// CreateMessage handles message creation.
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

		msg, err := h.Service.CreateMessage(body.UserID, body.ChatID, body.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Increment popularity for the chat.
		if err := h.Cache.IncrChatPopularity(body.ChatID, 1.0); err != nil {
			log.Println("Redis error:", err)
		}

		// Publish Kafka event for new message.
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
		if err := h.Publisher.PublishNewMessage(eventBytes); err != nil {
			log.Println("Kafka publish error:", err)
		}

		res.Json(w, msg, http.StatusCreated)
	}
}

// ListMessages returns all messages.
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

// ListMessagesByChat returns messages for a given chat.
func (h *Handler) ListMessagesByChat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		chatIDStr := r.URL.Query().Get("chat_id")
		if chatIDStr == "" {
			http.Error(w, "missing chat_id", http.StatusBadRequest)
			return
		}
		chatID, err := strconv.ParseUint(chatIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid chat_id", http.StatusBadRequest)
			return
		}
		msgs, err := h.Service.ListMessagesByChat(uint(chatID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, msgs, http.StatusOK)
	}
}

// UpdateMessage updates the content of an existing message.
func (h *Handler) UpdateMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			MessageID uint   `json:"message_id"`
			Content   string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := h.Service.UpdateMessage(body.MessageID, body.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Json(w, map[string]string{"status": "ok"}, http.StatusOK)
	}
}

// DeleteMessage removes a message.
func (h *Handler) DeleteMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete && r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			MessageID uint `json:"message_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := h.Service.DeleteMessage(body.MessageID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Json(w, map[string]string{"status": "deleted"}, http.StatusOK)
	}
}

// UploadMedia handles file uploads to the MediaService.
func (h *Handler) UploadMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "failed to parse form", http.StatusBadRequest)
			return
		}
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "failed to read file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		mediaURL, err := h.uploadToMediaService(file, fileHeader.Filename)
		if err != nil {
			http.Error(w, "failed to upload media", http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]string{"media_url": mediaURL}, http.StatusOK)
	}
}

// ListPopularChats returns the top popular chats.
func (h *Handler) ListPopularChats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limitStr := r.URL.Query().Get("limit")
		if limitStr == "" {
			limitStr = "10"
		}
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		chats, err := h.Cache.GetTopPopularChats(int64(limit))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, chats, http.StatusOK)
	}
}

// uploadToMediaService sends the file to the MediaService.
func (h *Handler) uploadToMediaService(file io.Reader, fileName string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(fileName))
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(part, file); err != nil {
		return "", err
	}
	writer.Close()

	url := fmt.Sprintf("%s/upload", h.Config.MediaServiceURL)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("media service error: %s", resp.Status)
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.URL, nil
}
