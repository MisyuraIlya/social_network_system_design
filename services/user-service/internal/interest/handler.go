package interest

import (
	"net/http"
	"strconv"

	"users-service/internal/shared/httpx"
	"users-service/internal/shared/validate"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

type CreateReq struct {
	Name string `json:"name" validate:"required"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	_, shardID, err := httpx.UserFromCtx(r)
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
	it, err := h.svc.Create(shardID, in.Name)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"id": it.ID, "name": it.Name}, http.StatusCreated)
	return nil
}

func (h *Handler) Attach(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	id, _ := strconv.ParseUint(r.PathValue("interest_id"), 10, 64)
	if id == 0 {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Attach(uid, id); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) Detach(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	id, _ := strconv.ParseUint(r.PathValue("interest_id"), 10, 64)
	if id == 0 {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Detach(uid, id); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.List(uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}
