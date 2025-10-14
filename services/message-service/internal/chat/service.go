package chat

import (
	"context"
	"message-service/internal/redisx"
)

type Service interface {
	Create(owner string, in CreateReq) (*Chat, error)
	GetByID(chatID int64) (*Chat, error)
	AddUser(chatID int64, userID string) error
	Join(chatID int64, userID string) error
	Leave(chatID int64, userID string) error
	ListMine(userID string, limit, offset int) ([]Chat, error)
	IncPopular(ctx context.Context, chatID int64)
	TopPopular(ctx context.Context, n int64) ([]int64, error)
}

type service struct {
	repo Repository
	rds  *redisx.Client
}

func NewService(r Repository, rds *redisx.Client) Service {
	return &service{repo: r, rds: rds}
}

func (s *service) Create(owner string, in CreateReq) (*Chat, error) {
	return s.repo.Create(owner, in.Name, in.Members)
}
func (s *service) GetByID(chatID int64) (*Chat, error) { return s.repo.GetByID(chatID) }
func (s *service) AddUser(chatID int64, userID string) error {
	return s.repo.AddUser(chatID, userID, "member")
}
func (s *service) Join(chatID int64, userID string) error {
	return s.repo.AddUser(chatID, userID, "member")
}
func (s *service) Leave(chatID int64, userID string) error { return s.repo.RemoveUser(chatID, userID) }
func (s *service) ListMine(userID string, limit, offset int) ([]Chat, error) {
	return s.repo.ListByUser(userID, limit, offset)
}
func (s *service) IncPopular(ctx context.Context, chatID int64) { s.rds.IncPopular(ctx, chatID) }
func (s *service) TopPopular(ctx context.Context, n int64) ([]int64, error) {
	return s.rds.TopPopular(ctx, n)
}
