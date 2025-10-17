package media

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"media-service/internal/shared/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	uid, _ := httpx.UserFromCtx(r) // optional
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "file required"}, http.StatusBadRequest)
		return
	}
	defer file.Close()

	prefix := r.FormValue("prefix")
	key := h.svc.BuildKey(prefix, header.Filename, uid)
	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	b, _ := io.ReadAll(file)
	if err := h.svc.s3.Put(r.Context(), key, ct, b); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	url, _ := h.svc.s3.PresignGet(r.Context(), key, 15*time.Minute)
	httpx.WriteJSON(w, map[string]any{
		"key":              key,
		"contentType":      ct,
		"url":              url.String(),
		"required_headers": map[string]string{"Content-Type": ct},
	}, http.StatusCreated)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httpx.WriteJSON(w, map[string]any{"error": "missing key"}, http.StatusBadRequest)
		return
	}
	if err := h.svc.s3.Remove(r.Context(), key); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RedirectToSignedGet(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.NotFound(w, r)
		return
	}
	ttl := 5 * time.Minute
	if s := r.URL.Query().Get("ttl"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 3600 {
			ttl = time.Duration(n) * time.Second
		}
	}
	u, err := h.svc.s3.PresignGet(r.Context(), key, ttl)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}

func (h *Handler) PresignPut(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Key         string `json:"key"`
		Prefix      string `json:"prefix"`
		FileName    string `json:"file_name"`
		ContentType string `json:"content_type"`
		ExpirySec   int    `json:"expiry_seconds"`
	}
	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "invalid json"}, http.StatusBadRequest)
		return
	}
	if body.Key == "" {
		uid, _ := httpx.UserFromCtx(r)
		body.Key = h.svc.BuildKey(strings.Trim(body.Prefix, "/"), body.FileName, uid)
	}
	if body.ExpirySec <= 0 || body.ExpirySec > 3600 {
		body.ExpirySec = 900
	}
	u, err := h.svc.s3.PresignPut(r.Context(), body.Key, time.Duration(body.ExpirySec)*time.Second, body.ContentType)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	httpx.WriteJSON(w, map[string]any{
		"key":              body.Key,
		"url":              u.String(),
		"ttl":              body.ExpirySec,
		"method":           "PUT",
		"required_headers": map[string]string{"Content-Type": body.ContentType},
	}, http.StatusOK)
}
