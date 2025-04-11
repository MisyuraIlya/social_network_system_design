package res

import (
	"encoding/json"
	"net/http"
)

// Json writes a JSON response with the given status code.
func Json(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
