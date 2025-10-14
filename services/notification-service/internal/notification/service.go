package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, userID string, kind Kind, title, body string, meta map[string]any) (Notification, error)
	List(ctx context.Context, userID string, limit int64) ([]Notification, error)
	MarkRead(ctx context.Context, userID, notifID string) error
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Create(ctx context.Context, userID string, kind Kind, title, body string, meta map[string]any) (Notification, error) {
	n := Notification{
		ID:        uuid.NewString(),
		UserID:    userID,
		Kind:      kind,
		Title:     title,
		Body:      body,
		Meta:      meta,
		CreatedAt: time.Now().UTC(),
	}
	return n, s.repo.Push(ctx, n)
}

func (s *service) List(ctx context.Context, userID string, limit int64) ([]Notification, error) {
	return s.repo.List(ctx, userID, limit)
}

func (s *service) MarkRead(ctx context.Context, userID, notifID string) error {
	return s.repo.MarkRead(ctx, userID, notifID)
}
