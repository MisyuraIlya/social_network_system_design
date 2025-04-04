package comments

import (
	"time"
)

type Service interface {
	CreateComment(userID, postID uint, name, text string) error
	GetComments(postID uint) ([]Comment, error)
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) CreateComment(userID, postID uint, name, text string) error {
	c := &Comment{
		UserID:    userID,
		PostID:    postID,
		Name:      name,
		Text:      text,
		CreatedAt: time.Now(),
	}
	return s.repo.Create(c)
}

func (s *service) GetComments(postID uint) ([]Comment, error) {
	return s.repo.GetAllByPostID(postID)
}
