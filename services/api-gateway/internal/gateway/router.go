package gateway

import (
	"net/http"

	"api-gateway/configs"
)

func InitRoutes(cfg *configs.Config) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("API Gateway OK"))
	})

	// Auth validator
	auth := func(r *http.Request) error {
		token, err := ExtractToken(r)
		if err != nil {
			return err
		}
		return ValidateToken(token, cfg.UserServiceURL)
	}

	// OPEN ROUTES
	mux.HandleFunc("/auth/", openProxyHandler(cfg.UserServiceURL))

	// SECURE ROUTES
	mux.Handle("/users/", secureProxyHandler(cfg.UserServiceURL, auth))
	mux.Handle("/posts/", secureProxyHandler(cfg.PostServiceURL, auth))
	mux.Handle("/messages/", secureProxyHandler(cfg.MessageServiceURL, auth))
	mux.Handle("/media/", secureProxyHandler(cfg.MediaServiceURL, auth))
	mux.Handle("/feed", secureProxyHandler(cfg.FeedServiceURL, auth))
	mux.Handle("/feed/", secureProxyHandler(cfg.FeedServiceURL, auth))
	mux.Handle("/feedback/", secureProxyHandler(cfg.FeedbackServiceURL, auth))

	// Fallback
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "API Gateway - endpoint not found", http.StatusNotFound)
	})

	return mux
}
