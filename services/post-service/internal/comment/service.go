package comment

import (
	"time"

	"post-service/internal/shared/validate"
)

type Service interface {
	Create(uid string, in CreateReq) (*Comment, error)
	ListByPost(postID uint64, limit, offset int) ([]Comment, error)
	Like(uid string, commentID uint64) error
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Create(uid string, in CreateReq) (*Comment, error) {
	if err := validate.Struct(in); err != nil {
		return nil, err
	}
	return s.repo.Create(&Comment{
		UserID: uid, PostID: in.PostID, Name: in.Name, Text: in.Text,
		CreatedAt: time.Now(),
	})
}

func (s *service) ListByPost(postID uint64, limit, offset int) ([]Comment, error) {
	return s.repo.ListByPost(postID, limit, offset)
}

func (s *service) Like(uid string, commentID uint64) error {
	_ = uid
	return s.repo.IncLike(commentID)
}
