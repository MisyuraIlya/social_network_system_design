package media

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type Handler struct {
	service MediaService
}

func NewHandler(service MediaService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) InitRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.healthCheck)
	mux.HandleFunc("/media/upload", h.uploadFile)
	return mux
}

func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		return
	}

	err := r.ParseMultipartForm(10 << 20) // limit memory usage
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to parse multipart form"})
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get file from form"})
		return
	}

	ctx := context.Background()
	mediaFile, err := h.service.Upload(ctx, file, fileHeader)
	if err != nil {
		log.Printf("Upload error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Upload failed"})
		return
	}

	response := UploadResponse{
		Message:  "File uploaded successfully",
		FileName: mediaFile.FileName,
		URL:      mediaFile.URL,
	}

	if strings.TrimSpace(response.URL) == "" {
		response.URL = "No URL generated"
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
