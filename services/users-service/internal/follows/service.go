package follows

import "time"

type Service interface {
	CreateFollow(userID, followedID int) error
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) CreateFollow(userID, followedID int) error {
	f := Follow{
		UserID:     userID,
		FollowedID: followedID,
		CreatedAt:  time.Now(),
	}
	return s.repo.CreateFollow(&f)
}
