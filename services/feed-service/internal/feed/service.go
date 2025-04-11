package feed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Service interface {
	CreateFeedItem(ctx context.Context, req CreateFeedRequest) (*CreateFeedResponse, error)
	GetFeed(ctx context.Context, userID string, page, pageSize int) ([]FeedItem, error)
	ConsumeNewPosts(ctx context.Context, message []byte) error
}

type service struct {
	repo       Repository
	userSvcURL string
	postSvcURL string
	httpClient *http.Client
}

type Friend struct {
	UserID    int       `json:"UserID"`
	FriendID  int       `json:"FriendID"`
	CreatedAt time.Time `json:"CreatedAt"`
}

func NewService(repo Repository, userSvcURL, postSvcURL string) Service {
	return &service{
		repo:       repo,
		userSvcURL: userSvcURL,
		postSvcURL: postSvcURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *service) CreateFeedItem(ctx context.Context, req CreateFeedRequest) (*CreateFeedResponse, error) {
	if req.UserID == "" || req.Content == "" {
		return nil, errors.New("user_id and content cannot be empty")
	}

	if err := s.validateUser(ctx, req.UserID); err != nil {
		return nil, fmt.Errorf("user validation failed: %w", err)
	}

	item := FeedItem{
		UserID:    req.UserID,
		PostID:    req.PostID,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveFeedItemWithLimit(item, 10); err != nil {
		return nil, err
	}

	return &CreateFeedResponse{Message: "Feed item created successfully"}, nil
}

func (s *service) GetFeed(ctx context.Context, userID string, page, pageSize int) ([]FeedItem, error) {
	if userID == "" {
		return nil, errors.New("user_id cannot be empty")
	}

	if page <= 1 {
		return s.repo.GetFeedItemsByUserID(userID)
	}
	friendIDs, err := s.fetchUserFriends(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetch friends error: %w", err)
	}

	friendIDs = append(friendIDs, userID)
	posts, err := s.fetchPostsByUserIDs(ctx, friendIDs, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("fetch posts error: %w", err)
	}

	var items []FeedItem
	for _, post := range posts {
		items = append(items, FeedItem{
			UserID:    fmt.Sprintf("%d", post.UserID),
			PostID:    fmt.Sprintf("%d", post.ID),
			Content:   post.Description,
			CreatedAt: post.CreatedAt,
		})
	}

	return items, nil
}

func (s *service) ConsumeNewPosts(ctx context.Context, message []byte) error {
	var post PostMessage
	if err := json.Unmarshal(message, &post); err != nil {
		log.Printf("Kafka message unmarshal error: %v", err)
		return err
	}

	item := FeedItem{
		UserID:    fmt.Sprintf("%d", post.UserID),
		PostID:    fmt.Sprintf("%d", post.ID),
		Content:   post.Description,
		CreatedAt: post.CreatedAt,
	}

	if err := s.repo.SaveFeedItemWithLimit(item, 10); err != nil {
		log.Printf("Error saving feed item: %v", err)
		return err
	}

	log.Printf("Processed new post ID %d for user %d", post.ID, post.UserID)
	return nil
}

func (s *service) validateUser(ctx context.Context, userID string) error {
	resp, err := s.httpClient.Get(fmt.Sprintf("%s/users/%s", s.userSvcURL, userID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("user service error: %s", string(body))
	}
	return nil
}

func (s *service) fetchUserFriends(ctx context.Context, userID string) ([]string, error) {
	resp, err := s.httpClient.Get(fmt.Sprintf("%s/users/%s/friends", s.userSvcURL, userID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("users service error: %s", string(body))
	}

	var friends []Friend
	if err := json.NewDecoder(resp.Body).Decode(&friends); err != nil {
		return nil, err
	}

	ids := make([]string, len(friends))
	for i, friend := range friends {
		ids[i] = strconv.Itoa(friend.UserID)
	}
	return ids, nil
}

func (s *service) fetchPostsByUserIDs(ctx context.Context, userIDs []string, page, pageSize int) ([]PostMessage, error) {
	url := fmt.Sprintf("%s/posts?user_ids=%s&page=%d&page_size=%d", s.postSvcURL, strings.Join(userIDs, ","), page, pageSize)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("post service error: %s", string(body))
	}

	var posts []PostMessage
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}
	return posts, nil
}
