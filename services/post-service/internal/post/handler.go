package post

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"post-service/internal/feedback"
	"post-service/internal/shared/httpx"
	"post-service/internal/shared/validate"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[CreateReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	p, err := h.svc.Create(uid, in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, p, http.StatusCreated)
	return nil
}

func (h *Handler) UploadAndCreate(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	if err := r.ParseMultipartForm(20 << 20); err != nil { // 20MB
		return err
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer file.Close()

	description := strings.TrimSpace(r.FormValue("description"))
	tags := strings.Split(strings.TrimSpace(r.FormValue("tags")), ",")
	if len(tags) == 1 && tags[0] == "" {
		tags = nil
	}

	p, err := h.svc.UploadAndCreate(uid, hdr.Filename, file, description, tags)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, p, http.StatusCreated)
	return nil
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	p, err := h.svc.GetByID(id)
	if err != nil {
		return err
	}

	// Enrich counts from feedback-service
	ctx, cancel := context.WithTimeout(r.Context(), feedback.DefaultTimeout)
	defer cancel()
	fb := feedback.NewClient("")
	likes, comments, _ := fb.GetCounts(ctx, p.ID)

	out := map[string]any{
		"id":          p.ID,
		"user_id":     p.UserID,
		"description": p.Description,
		"media":       p.MediaURL,
		"views":       p.Views,
		"likes":       likes,
		"comments":    comments,
		"created_at":  p.CreatedAt,
		"updated_at":  p.UpdatedAt,
	}
	httpx.WriteJSON(w, out, http.StatusOK)
	return nil
}

func (h *Handler) ListByUser(w http.ResponseWriter, r *http.Request) error {
	uid := r.PathValue("user_id")
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.ListByUser(uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) AddView(w http.ResponseWriter, r *http.Request) error {
	id, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	if err := h.svc.AddView(id); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}
