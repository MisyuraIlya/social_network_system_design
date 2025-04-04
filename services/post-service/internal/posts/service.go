package posts

import (
	"errors"
	"time"
)

type Service interface {
	Create(userID uint, description, media string) error
	GetAll() ([]Post, error)
	GetByID(id uint) (*Post, error)
	Update(id uint, description string) error
	Delete(id uint) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
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
	return s.repo.Create(p)
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
		return errors.New("post not found")
	}
	p.Description = description
	return s.repo.Update(p)
}

func (s *service) Delete(id uint) error {
	return s.repo.Delete(id)
}
