package comment

import (
	"feedback-gateway/internal/like"
	"feedback-gateway/internal/shared/httpx"
	"feedback-gateway/internal/shared/validate"
	"net/http"
	"strconv"
)

type Handler struct {
	svc     Service
	likeSvc like.Service
}

func NewHandler(s Service) *Handler                { return &Handler{svc: s} }
func (h *Handler) WithLikeService(ls like.Service) { h.likeSvc = ls }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	in, err := httpx.Decode[CreateReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	c, err := h.svc.Create(uid, pid, in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, c, http.StatusCreated)
	return nil
}

func (h *Handler) DeleteMine(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseUint(r.PathValue("comment_id"), 10, 64)
	if err := h.svc.DeleteMine(uid, cid); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) ListByPost(w http.ResponseWriter, r *http.Request) error {
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.ListByPost(pid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{
		"items": items, "limit": limit, "offset": offset,
	}, http.StatusOK)
	return nil
}

func (h *Handler) GetCounts(w http.ResponseWriter, r *http.Request) error {
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	cCount, err := h.svc.CommentCount(pid)
	if err != nil {
		return err
	}
	var lCount int64
	if h.likeSvc != nil {
		l, _, e := h.likeSvc.Get(pid, "")
		if e == nil {
			lCount = l
		}
	}
	httpx.WriteJSON(w, map[string]any{"post_id": pid, "likes": lCount, "comments": cCount}, http.StatusOK)
	return nil
}
