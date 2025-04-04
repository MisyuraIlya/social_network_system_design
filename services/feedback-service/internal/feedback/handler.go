package feedback

import (
	"feedback-service/configs"
	"feedback-service/pkg/req"
	"feedback-service/pkg/res"
	"net/http"
)

type Handler struct {
	*configs.Config
	*Service
}

type HandlerDeps struct {
	Config  *configs.Config
	Service *Service
}

func NewHandler(router *http.ServeMux, deps HandlerDeps) {
	h := &Handler{
		Config:  deps.Config,
		Service: deps.Service,
	}

	router.HandleFunc("/feedback/like", h.Like())
	router.HandleFunc("/feedback/comment", h.Comment())
}

func (h *Handler) Like() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[LikeRequest](&w, r)
		if err != nil {
			return
		}
		if err := h.Service.Like(*body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]string{"status": "liked"}, http.StatusCreated)
	}
}

func (h *Handler) Comment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[CommentRequest](&w, r)
		if err != nil {
			return
		}
		if err := h.Service.Comment(*body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]string{"status": "commented"}, http.StatusCreated)
	}
}
