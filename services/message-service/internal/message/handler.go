package message

import (
	"io"
	"net/http"
	"strconv"

	"message-service/internal/shared/httpx"
	"message-service/internal/shared/validate"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	in, err := httpx.Decode[SendReq](r)
	if err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	m, err := h.svc.Send(r.Context(), uid, in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, m, http.StatusCreated)
	return nil
}

func (h *Handler) UploadAndSend(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return err
	}
	f, fh, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer f.Close()
	data, _ := io.ReadAll(f)
	chatID, _ := strconv.ParseInt(r.FormValue("chat_id"), 10, 64)
	text := r.FormValue("text")
	bearer := httpx.BearerToken(r)
	m, err := h.svc.SendWithUpload(r.Context(), uid, chatID, fh.Filename, data, text, bearer)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, m, http.StatusCreated)
	return nil
}

func (h *Handler) ListByChat(w http.ResponseWriter, r *http.Request) error {
	_, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	limit := qint(r, "limit", 50)
	offset := qint(r, "offset", 0)
	items, err := h.svc.ListByChat(cid, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "limit": limit, "offset": offset}, http.StatusOK)
	return nil
}

func (h *Handler) MarkSeen(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	mid, _ := strconv.ParseInt(r.PathValue("message_id"), 10, 64)
	if err := h.svc.MarkSeen(mid, uid); err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
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
