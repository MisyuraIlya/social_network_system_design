package social

import (
	"net/http"

	"users-service/internal/shared/httpx"
	"users-service/internal/shared/validate"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Follow(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	target := r.PathValue("target_id")
	if target == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Follow(uid, target); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) Unfollow(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	target := r.PathValue("target_id")
	if target == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Unfollow(uid, target); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) ListFollowing(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.ListFollowing(uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) Befriend(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	friend := r.PathValue("friend_id")
	if friend == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Befriend(uid, friend); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) Unfriend(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	friend := r.PathValue("friend_id")
	if friend == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.Unfriend(uid, friend); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.ListFriends(uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

type relReq struct {
	RelatedID string `json:"related_id" validate:"required"`
	Type      int    `json:"type" validate:"required"`
}

func (h *Handler) CreateRelationship(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[relReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	if err := h.svc.CreateRelationship(uid, in.RelatedID, in.Type); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) DeleteRelationship(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[relReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	if err := h.svc.DeleteRelationship(uid, in.RelatedID, in.Type); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) ListRelationships(w http.ResponseWriter, r *http.Request) error {
	uid, _, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	typ := httpx.QueryInt(r, "type", 0) // 0 = any
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.ListRelationships(uid, typ, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{
		"items":  items,
		"type":   typ,
		"limit":  limit,
		"offset": offset,
	}, http.StatusOK)
	return nil
}
