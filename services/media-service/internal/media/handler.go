package media

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type MediaHandler struct {
	Service    IMediaService
	StorageDir string
}

func NewMediaHandler(svc IMediaService, storageDir string) *MediaHandler {
	return &MediaHandler{
		Service:    svc,
		StorageDir: storageDir,
	}
}

func RegisterRoutes(mux *http.ServeMux, handler *MediaHandler) {
	mux.HandleFunc("/media", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.UploadFile(w, r)
			return
		}
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/media/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 2 {
			http.Error(w, "Missing media ID", http.StatusBadRequest)
			return
		}
		handler.GetFile(w, r, pathParts[1])
	})
}

func (h *MediaHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	media, err := h.Service.Upload(file, header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(media)
}

func (h *MediaHandler) GetFile(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid media ID", http.StatusBadRequest)
		return
	}
	media, err := h.Service.Get(uint(id))
	if err != nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}
	filePath := filepath.Join(h.StorageDir, media.FileName)
	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", media.ContentType)
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=%s", media.FileName),
	)
	http.ServeFile(w, r, filePath)
}
