package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"
)

type Service interface {
	GetAuthorFeed(ctx context.Context, authorID string, limit, offset int) ([]FeedEntry, error)
	GetHomeFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error)
	RebuildHomeFeed(ctx context.Context, userID, bearer string, limit int) error
}

type service struct {
	repo             Repository
	userSvcBase      string
	defaultFeedLimit int
	httpClient       *http.Client
}

type Option func(*service)

func WithUserServiceBase(base string) Option {
	return func(s *service) { s.userSvcBase = base }
}
func WithDefaultFeedLimit(n int) Option {
	return func(s *service) { s.defaultFeedLimit = n }
}

func NewService(r Repository, opts ...Option) Service {
	s := &service{
		repo:             r,
		userSvcBase:      envOr("USER_SERVICE_URL", "http://user-service:8081"),
		defaultFeedLimit: 100,
		httpClient:       &http.Client{Timeout: 5 * time.Second},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func (s *service) GetAuthorFeed(ctx context.Context, authorID string, limit, offset int) ([]FeedEntry, error) {
	return s.repo.GetAuthorFeed(ctx, authorID, limit, offset)
}

func (s *service) GetHomeFeed(ctx context.Context, userID string, limit, offset int) ([]FeedEntry, error) {
	return s.repo.GetHomeFeed(ctx, userID, limit, offset)
}

func (s *service) RebuildHomeFeed(ctx context.Context, userID, bearer string, limit int) error {
	if limit <= 0 {
		limit = s.defaultFeedLimit
	}

	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/relationships?type=1&limit=%d&offset=%d", s.userSvcBase, 5000, 0), nil)
	req.Header.Set("Authorization", "Bearer "+bearer)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("user-service call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("user-service: bad status %d", resp.StatusCode)
	}
	var rel struct {
		Items []string `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return err
	}

	all := make([]FeedEntry, 0, limit*2)
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	perAuthor := limit / 5
	if perAuthor < 10 {
		perAuthor = 10
	}
	if perAuthor > 100 {
		perAuthor = 100
	}
	for _, authorID := range rel.Items {
		ents, e := s.repo.GetAuthorFeed(ctx2, authorID, perAuthor, 0)
		if e == nil && len(ents) > 0 {
			all = append(all, ents...)
		}
	}

	sort.Slice(all, func(i, j int) bool { return all[i].Score > all[j].Score })
	if len(all) > limit {
		all = all[:limit]
	}
	return s.repo.StoreHomeFeed(ctx, userID, all)
}
