package profile

import (
	"net/http"

	"users-service/internal/shared/httpx"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[UpsertReq](r)
	if err != nil {
		return err
	}
	if err := h.svc.Upsert(uid, in); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}
func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) error {
	uid := r.PathValue("user_id")
	p, err := h.svc.GetPublic(uid)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, p, http.StatusOK)
	return nil
}
