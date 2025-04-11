package friends

import "time"

type Service interface {
	CreateFriend(userID, friendID int) error
	GetFriends(userID int) ([]Friend, error)
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

func (s *service) GetFriends(userID int) ([]Friend, error) {
	friends, err := s.repo.GetFriends(userID)
	if err != nil {
		return nil, err
	}
	return friends, nil
}
