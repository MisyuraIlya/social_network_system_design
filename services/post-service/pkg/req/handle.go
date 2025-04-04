// pkg/req/handle.go
package req

import (
	"net/http"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) error

func MultiHandle(methodMap map[string]HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fn, ok := methodMap[r.Method]
		if !ok {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Call the method-specific handler
		if err := fn(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
