package message

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type MessageHandler struct {
	Service IMessageService
}

func NewMessageHandler(svc IMessageService) *MessageHandler {
	return &MessageHandler{Service: svc}
}

func RegisterRoutes(mux *http.ServeMux, handler *MessageHandler) {
	mux.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.SendMessage(w, r)
		case http.MethodGet:
			handler.GetMessages(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
}

func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	msg, err := h.Service.SendMessage(req.DialogID, req.SenderID, req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := SendMessageResponse{
		ID:        msg.ID,
		DialogID:  msg.DialogID,
		SenderID:  msg.SenderID,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	dialogIDStr := r.URL.Query().Get("dialogId")
	if dialogIDStr == "" {
		http.Error(w, "dialogId is required", http.StatusBadRequest)
		return
	}

	dialogID, err := strconv.Atoi(dialogIDStr)
	if err != nil || dialogID <= 0 {
		http.Error(w, "invalid dialogId", http.StatusBadRequest)
		return
	}

	messages, err := h.Service.GetMessages(uint(dialogID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := make([]MessageData, 0, len(messages))
	for _, m := range messages {
		data = append(data, MessageData{
			ID:        m.ID,
			DialogID:  m.DialogID,
			SenderID:  m.SenderID,
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		})
	}

	resp := GetMessagesResponse{Messages: data}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
