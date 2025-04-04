package media

import "net/http"

type UploadResponse struct {
	Message  string `json:"message"`
	FileName string `json:"file_name"`
	URL      string `json:"url,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
}
