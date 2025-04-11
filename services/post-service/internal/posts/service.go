package posts

import (
	"context"
	"encoding/json"
	"time"

	"post-service/pkg/kafka"
)

type Service interface {
	Create(userID uint, description, media string) error
	GetAll() ([]Post, error)
	GetByID(id uint) (*Post, error)
	Update(id uint, description string) error
	Delete(id uint) error
}

type service struct {
	repo     Repository
	producer *kafka.Producer
}

func NewService(repo Repository, producer *kafka.Producer) Service {
	return &service{repo: repo, producer: producer}
}

func (s *service) Create(userID uint, description, media string) error {
	p := &Post{
		UserID:      userID,
		Description: description,
		Media:       media,
		Likes:       0,
		Views:       0,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.Create(p); err != nil {
		return err
	}

	msg, err := json.Marshal(p)
	if err != nil {
		return err
	}

	return s.producer.Publish(context.Background(), "post_created", msg)
}

func (s *service) GetAll() ([]Post, error) {
	return s.repo.GetAll()
}

func (s *service) GetByID(id uint) (*Post, error) {
	return s.repo.GetByID(id)
}

func (s *service) Update(id uint, description string) error {
	p, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if p == nil {
		return err
	}
	p.Description = description
	return s.repo.Update(p)
}

func (s *service) Delete(id uint) error {
	return s.repo.Delete(id)
}
