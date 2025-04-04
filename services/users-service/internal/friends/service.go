package friends

import "time"

type Service interface {
	CreateFriend(userID, friendID int) error
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) CreateFriend(userID, friendID int) error {
	f := Friend{
		UserID:    userID,
		FriendID:  friendID,
		CreatedAt: time.Now(),
	}
	return s.repo.CreateFriend(&f)
}
