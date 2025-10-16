package notification

import (
	"encoding/json"
	"net/http"
	"strconv"

	"notification-service/internal/shared/httpx"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return errUnauthorized("auth required")
	}

	if pathUID := r.PathValue("user_id"); pathUID != "" && pathUID != uid {
		return errUnauthorized("forbidden: cannot read other users' notifications")
	}

	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	items, err := h.svc.List(r.Context(), uid, limit)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"notifications": items}, http.StatusOK)
	return nil
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return errUnauthorized("auth required")
	}
	id := r.PathValue("id")
	if id == "" {
		return errBadReq("missing id")
	}
	if err := h.svc.MarkRead(r.Context(), uid, id); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
	return nil
}

func (h *Handler) CreateTest(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		UserID string         `json:"user_id"`
		Title  string         `json:"title"`
		Body   string         `json:"body"`
		Kind   Kind           `json:"kind"`
		Meta   map[string]any `json:"meta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errBadReq("bad json")
	}
	if req.Kind == "" {
		req.Kind = KindMessage
	}
	n, err := h.svc.Create(r.Context(), req.UserID, req.Kind, req.Title, req.Body, req.Meta)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, n, http.StatusCreated)
	return nil
}

type httpErr struct {
	msg  string
	code int
}

func (e httpErr) Error() string      { return e.msg }
func errBadReq(m string) error       { return httpErr{m, http.StatusBadRequest} }
func errUnauthorized(m string) error { return httpErr{m, http.StatusUnauthorized} }
