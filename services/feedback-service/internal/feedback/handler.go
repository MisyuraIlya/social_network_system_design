package feedback

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type FeedbackHandler struct {
	Service IFeedbackService
}

func NewFeedbackHandler(s IFeedbackService) *FeedbackHandler {
	return &FeedbackHandler{Service: s}
}

func RegisterRoutes(mux *http.ServeMux, h *FeedbackHandler) {
	mux.HandleFunc("/feedback/like", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.Like(w, r)
		case http.MethodDelete:
			h.Unlike(w, r)
		case http.MethodGet:
			h.GetLikeCount(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/feedback/comment", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.AddComment(w, r)
		case http.MethodGet:
			h.GetComments(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/feedback/comment/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 {
			http.Error(w, "missing comment ID", http.StatusBadRequest)
			return
		}
		cID, err := strconv.Atoi(parts[2])
		if err != nil {
			http.Error(w, "invalid comment ID", http.StatusBadRequest)
			return
		}
		h.DeleteCommentByID(w, r, uint(cID))
	})
}

func (h *FeedbackHandler) Like(w http.ResponseWriter, r *http.Request) {
	var req LikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	like, err := h.Service.LikePost(req.UserID, req.PostID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := LikeResponse{
		ID:     like.ID,
		UserID: like.UserID,
		PostID: like.PostID,
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *FeedbackHandler) Unlike(w http.ResponseWriter, r *http.Request) {
	var req LikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := h.Service.UnlikePost(req.UserID, req.PostID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"unliked"}`))
}

func (h *FeedbackHandler) GetLikeCount(w http.ResponseWriter, r *http.Request) {
	postIDStr := r.URL.Query().Get("postId")
	if postIDStr == "" {
		http.Error(w, "postId is required", http.StatusBadRequest)
		return
	}
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "invalid postId", http.StatusBadRequest)
		return
	}
	count, err := h.Service.CountLikes(uint(postID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := CountLikesResponse{
		PostID:    uint(postID),
		LikeCount: count,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *FeedbackHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	comment, err := h.Service.AddComment(req.UserID, req.PostID, req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := CommentResponse{
		ID:        comment.ID,
		UserID:    comment.UserID,
		PostID:    comment.PostID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *FeedbackHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	postIDStr := r.URL.Query().Get("postId")
	if postIDStr == "" {
		http.Error(w, "postId is required", http.StatusBadRequest)
		return
	}
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "invalid postId", http.StatusBadRequest)
		return
	}
	comments, err := h.Service.GetComments(uint(postID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		data = append(data, CommentResponse{
			ID:        c.ID,
			UserID:    c.UserID,
			PostID:    c.PostID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
		})
	}
	resp := GetCommentsResponse{Comments: data}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *FeedbackHandler) DeleteCommentByID(w http.ResponseWriter, r *http.Request, commentID uint) {
	err := h.Service.DeleteComment(commentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"deleted comment"}`))
}
