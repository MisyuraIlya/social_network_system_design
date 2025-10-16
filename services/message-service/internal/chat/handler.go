package chat

import (
	"net/http"
	"strconv"

	"message-service/internal/shared/httpx"
	"message-service/internal/shared/validate"
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

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	id, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	c, err := h.svc.GetByID(id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, c, http.StatusOK)
	return nil
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := qint(r, "limit", 50)
	offset := qint(r, "offset", 0)
	out, err := h.svc.ListMine(uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": out, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) Join(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	if cid == 0 {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Join(cid, uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, ok(), http.StatusOK)
	return nil
}

func (h *Handler) AddUser(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	uid := r.PathValue("user_id")
	if cid == 0 || uid == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.AddUser(cid, actorID, uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, ok(), http.StatusOK)
	return nil
}

func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	if cid == 0 {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Leave(cid, uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, ok(), http.StatusOK)
	return nil
}

func (h *Handler) Popular(w http.ResponseWriter, r *http.Request) error {
	top := qint(r, "top", 10)
	ids, err := h.svc.TopPopular(r.Context(), int64(top))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"chat_ids": ids}, http.StatusOK)
	return nil
}

func qint(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		return def
	}
	return n
}
func ok() map[string]string { return map[string]string{"status": "ok"} }
