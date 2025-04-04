package relationships

import (
	"encoding/json"
	"net/http"
	"users-service/configs"
	"users-service/pkg/res"
)

type HandlerDeps struct {
	*configs.Config
	Service Service
}

type Handler struct {
	*configs.Config
	Service Service
}

func NewHandler(router *http.ServeMux, deps HandlerDeps) {
	h := &Handler{
		Config:  deps.Config,
		Service: deps.Service,
	}
	router.HandleFunc("/users/relationships/create", h.CreateRelationship())
}

func (h *Handler) CreateRelationship() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			UserID           int `json:"user_id"`
			RelatedID        int `json:"related_id"`
			RelationshipType int `json:"relationship_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := h.Service.CreateRelationship(body.UserID, body.RelatedID, body.RelationshipType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]string{"status": "relationship created"}, http.StatusOK)
	}
}
