package feed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Service is the business logic layer for the feed.
type Service interface {
	CreateFeedItem(ctx context.Context, req CreateFeedRequest) (*CreateFeedResponse, error)
	GetFeed(ctx context.Context, userID string) ([]FeedItem, error)
	ConsumeNewPosts(ctx context.Context, message []byte) error // Example of Kafka consumer
}

// service implements Service.
type service struct {
	repo       Repository
	userSvcURL string
	httpClient *http.Client
}

// NewService creates a new feed service instance.
func NewService(repo Repository, userSvcURL string) Service {
	return &service{
		repo:       repo,
		userSvcURL: userSvcURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// CreateFeedItem handles creation of a feed item, possibly after validation with user service.
func (s *service) CreateFeedItem(ctx context.Context, req CreateFeedRequest) (*CreateFeedResponse, error) {
	if req.UserID == "" {
		return nil, errors.New("user_id cannot be empty")
	}
	if req.Content == "" {
		return nil, errors.New("content cannot be empty")
	}

	// Example: Validate user from user service
	err := s.validateUser(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user validation failed: %w", err)
	}

	feedItem := FeedItem{
		UserID:    req.UserID,
		PostID:    req.PostID,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}
	if err := s.repo.SaveFeedItem(feedItem); err != nil {
		return nil, err
	}

	return &CreateFeedResponse{Message: "Feed item created successfully"}, nil
}

// GetFeed retrieves feed items for a specific user.
func (s *service) GetFeed(ctx context.Context, userID string) ([]FeedItem, error) {
	if userID == "" {
		return nil, errors.New("user_id cannot be empty")
	}
	items, err := s.repo.GetFeedItemsByUserID(userID)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// ConsumeNewPosts is an example function that you might call from your Kafka consumer
// to process new post messages and store them in Redis.
func (s *service) ConsumeNewPosts(ctx context.Context, message []byte) error {
	var req CreateFeedRequest
	if err := json.Unmarshal(message, &req); err != nil {
		log.Printf("Failed to unmarshal Kafka message: %v", err)
		return err
	}
	_, err := s.CreateFeedItem(ctx, req)
	return err
}

// validateUser calls the user service to confirm the user exists.
func (s *service) validateUser(ctx context.Context, userID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.userSvcURL+"/users/"+userID, nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("User validation failed, status %d, body: %s", resp.StatusCode, string(bodyBytes))
		return errors.New("user not found or user service error")
	}
	return nil
}
