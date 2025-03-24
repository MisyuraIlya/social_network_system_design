package gateway

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func newReverseProxy(targetHost string) http.Handler {
	target, _ := url.Parse(targetHost)

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		return nil
	}

	return proxy
}

func stripPathPrefix(h http.Handler, prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, prefix) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		}
		h.ServeHTTP(w, r)
	})
}
