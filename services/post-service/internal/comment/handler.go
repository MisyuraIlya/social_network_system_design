package comment

import (
	"net/http"
	"strconv"

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
	c, err := h.svc.Create(uid, in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, c, http.StatusCreated)
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
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) Like(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	id, _ := strconv.ParseUint(r.PathValue("comment_id"), 10, 64)
	if err := h.svc.Like(uid, id); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}
