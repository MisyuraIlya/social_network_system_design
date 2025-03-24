package gateway

import (
	"net/http"

	"api-gateway/configs"
)

type GatewayHandler struct {
	cfg *configs.Config
}

func NewGatewayHandler(cfg *configs.Config) *GatewayHandler {
	return &GatewayHandler{cfg: cfg}
}

func (g *GatewayHandler) RegisterRoutes(mux *http.ServeMux) {
	userProxy := newReverseProxy(g.cfg.UserServiceURL)
	mux.Handle("/users", userProxy)
	mux.Handle("/users/", stripPathPrefix(userProxy, "/users"))

	postProxy := newReverseProxy(g.cfg.PostServiceURL)
	mux.Handle("/posts", postProxy)
	mux.Handle("/posts/", stripPathPrefix(postProxy, "/posts"))

	messageProxy := newReverseProxy(g.cfg.MessageServiceURL)
	mux.Handle("/messages", messageProxy)
	mux.Handle("/messages/", stripPathPrefix(messageProxy, "/messages"))

	mediaProxy := newReverseProxy(g.cfg.MediaServiceURL)
	mux.Handle("/media", mediaProxy)
	mux.Handle("/media/", stripPathPrefix(mediaProxy, "/media"))

	feedProxy := newReverseProxy(g.cfg.FeedServiceURL)
	mux.Handle("/feed", feedProxy)
	mux.Handle("/feed/", stripPathPrefix(feedProxy, "/feed"))

	feedbackProxy := newReverseProxy(g.cfg.FeedbackServiceURL)
	mux.Handle("/feedback", feedbackProxy)
	mux.Handle("/feedback/", stripPathPrefix(feedbackProxy, "/feedback"))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "API Gateway - endpoint not found", http.StatusNotFound)
	})
}
