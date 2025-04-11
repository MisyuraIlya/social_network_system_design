package posts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"post-service/configs"
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
	repo       Repository
	producer   *kafka.Producer
	config     *configs.Config
	httpClient *http.Client
}

func NewService(repo Repository, producer *kafka.Producer, config *configs.Config) Service {
	return &service{
		repo:       repo,
		producer:   producer,
		config:     config,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
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
	// get user friends
	friendIds, err := s.getUserFriends(userID)
	fmt.Printf("Friend IDs: %v\n", friendIds)

	//
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

func (s *service) getUserFriends(UserId uint) ([]uint, error) {
	url := fmt.Sprintf("%s/users/%d/friends", s.config.UsersServiceURL, UserId)
	response, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get friends: %s", response.Status)
	}

	var friends []UserFriends
	if err := json.NewDecoder(response.Body).Decode(&friends); err != nil {
		return nil, err
	}

	ids := []uint{}
	for _, friend := range friends {
		ids = append(ids, friend.FriendID)
	}
	return ids, nil

}
