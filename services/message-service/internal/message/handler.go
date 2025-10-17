package message

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"message-service/internal/idem"
	"message-service/internal/shared/httpx"
	"message-service/internal/shared/validate"
)

type Handler struct {
	svc  Service
	idem idem.Store
}

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) WithIdem(s idem.Store) *Handler {
	h.idem = s
	return h
}

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

	if h.idem != nil {
		if key := r.Header.Get("Idempotency-Key"); key != "" {
			ok, e := h.idem.PutNX(r.Context(), "send:"+uid+":"+strconv.FormatInt(in.ChatID, 10)+":"+key, 24*time.Hour)
			if e != nil {
				return e
			}
			if !ok {
				httpx.WriteJSON(w, map[string]any{"error": "duplicate request"}, http.StatusConflict)
				return nil
			}
		}
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

	if h.idem != nil {
		if key := r.Header.Get("Idempotency-Key"); key != "" {
			ok, e := h.idem.PutNX(r.Context(), "send-upload:"+uid+":"+strconv.FormatInt(chatID, 10)+":"+key, 24*time.Hour)
			if e != nil {
				return e
			}
			if !ok {
				httpx.WriteJSON(w, map[string]any{"error": "duplicate request"}, http.StatusConflict)
				return nil
			}
		}
	}

	m, err := h.svc.SendWithUpload(r.Context(), uid, chatID, fh.Filename, data, text, bearer)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, m, http.StatusCreated)
	return nil
}

func (h *Handler) ListByChat(w http.ResponseWriter, r *http.Request) error {
	uid, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	cid, _ := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	limit := qint(r, "limit", 50)
	offset := qint(r, "offset", 0)

	items, err := h.svc.ListByChat(uid, cid, limit, offset)
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
