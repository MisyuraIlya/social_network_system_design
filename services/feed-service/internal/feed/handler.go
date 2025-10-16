package feed

import (
	"net/http"

	"feed-service/internal/shared/httpx"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

// Public: feed by author
func (h *Handler) GetAuthorFeed(w http.ResponseWriter, r *http.Request) error {
	uid := r.PathValue("user_id")
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.GetAuthorFeed(r.Context(), uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

// Protected: home feed of the current user
func (h *Handler) GetHomeFeed(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.GetHomeFeed(r.Context(), uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) RebuildHomeFeed(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	bearer := httpx.BearerToken(r)
	limit := httpx.QueryInt(r, "limit", 100)
	if err := h.svc.RebuildHomeFeed(r.Context(), uid, bearer, limit); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

// ---------- Celebrities ----------

// Public: feed by celebrity user_id
func (h *Handler) GetCelebrityFeed(w http.ResponseWriter, r *http.Request) error {
	uid := r.PathValue("user_id")
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	items, err := h.svc.GetCelebrityFeed(r.Context(), uid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

// Public: list celebrity IDs (could be cached by clients)
func (h *Handler) ListCelebrities(w http.ResponseWriter, r *http.Request) error {
	ids, err := h.svc.ListCelebrities(r.Context())
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": ids}, http.StatusOK)
	return nil
}

// Protected: promote a user to celebrity set
func (h *Handler) PromoteCelebrity(w http.ResponseWriter, r *http.Request) error {
	_, err := httpx.UserFromCtx(r) // simple auth gate; tighten to admin if you add roles later
	if err != nil {
		return err
	}
	uid := r.PathValue("user_id")
	if uid == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.PromoteCelebrity(r.Context(), uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

// Protected: demote a user from celebrity set
func (h *Handler) DemoteCelebrity(w http.ResponseWriter, r *http.Request) error {
	_, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	uid := r.PathValue("user_id")
	if uid == "" {
		return httpx.ErrUnauthorized
	}
	if err := h.svc.DemoteCelebrity(r.Context(), uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}
