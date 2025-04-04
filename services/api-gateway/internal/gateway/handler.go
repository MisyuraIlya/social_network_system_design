package gateway

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func openProxyHandler(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetURL, err := url.Parse(target)
		if err != nil {
			http.Error(w, "Bad target URL", http.StatusInternalServerError)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Log errors from backend services
		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			log.Printf("Proxy error: %v", err)
			http.Error(rw, "Proxy error: "+err.Error(), http.StatusBadGateway)
		}

		proxy.ServeHTTP(w, r)
	}
}

func secureProxyHandler(target string, validateFunc func(r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := validateFunc(r); err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		openProxyHandler(target)(w, r)
	}
}
