package like

import (
	"feedback-gateway/internal/shared/httpx"
	"net/http"
	"strconv"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Like(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	count, err := h.svc.Like(uid, pid)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"post_id": pid, "likes": count, "liked_by_me": true}, http.StatusOK)
	return nil
}

func (h *Handler) Unlike(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	count, err := h.svc.Unlike(uid, pid)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"post_id": pid, "likes": count, "liked_by_me": false}, http.StatusOK)
	return nil
}

func (h *Handler) GetLikes(w http.ResponseWriter, r *http.Request) error {
	uid, _ := httpx.UserFromCtx(r)
	pid, _ := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	count, liked, err := h.svc.Get(pid, uid)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"post_id": pid, "likes": count, "liked_by_me": liked}, http.StatusOK)
	return nil
}
