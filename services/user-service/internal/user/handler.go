package user

import (
	"net/http"

	"users-service/internal/shared/httpx"
	"users-service/internal/shared/jwt"
	"users-service/internal/shared/validate"
)

type Handler struct{ svc Service }

func NewHandler(s Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) error {
	body, err := httpx.Decode[RegisterReq](r)
	if err != nil {
		return err
	}
	if err = validate.Struct(body); err != nil {
		return err
	}
	u, err := h.svc.Register(body.Email, body.Password, body.Name)
	if err != nil {
		return err
	}
	token, _ := jwt.Make(u.UserID, u.ShardID)
	httpx.WriteJSON(w, map[string]any{
		"user_id": u.UserID, "name": u.Name, "email": u.Email, "access_token": token,
	}, http.StatusCreated)
	return nil
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	body, err := httpx.Decode[LoginReq](r)
	if err != nil {
		return err
	}
	if err = validate.Struct(body); err != nil {
		return err
	}
	u, err := h.svc.Login(body.Email, body.Password)
	if err != nil {
		return err
	}
	token, _ := jwt.Make(u.UserID, u.ShardID)
	httpx.WriteJSON(w, map[string]any{
		"message": "login successful", "user_id": u.UserID, "name": u.Name, "email": u.Email, "access_token": token,
	}, http.StatusOK)
	return nil
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	uid := r.PathValue("user_id")
	u, err := h.svc.GetByUserID(uid)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, u, http.StatusOK)
	return nil
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) error {
	_, shardID, err := httpx.UserFromCtx(r)
	if err != nil {
		return err
	}
	limit := httpx.QueryInt(r, "limit", 50)
	offset := httpx.QueryInt(r, "offset", 0)
	users, err := h.svc.ListMine(shardID, limit, offset)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, map[string]any{"shard_id": shardID, "limit": limit, "offset": offset, "items": users}, http.StatusOK)
	return nil
}
