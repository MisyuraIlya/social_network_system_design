package feedback

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const DefaultTimeout = 3 * time.Second

type Client struct {
	base string
	hc   *http.Client
}

func NewClient(base string) *Client {
	if base == "" {
		base = getenv("FEEDBACK_SERVICE_URL", "http://feedback-service:8084")
	}
	return &Client{
		base: base,
		hc:   &http.Client{Timeout: DefaultTimeout},
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func (c *Client) GetCounts(ctx context.Context, postID uint64) (likes int64, comments int64, err error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/posts/%d/counts", c.base, postID), nil)
	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return 0, 0, fmt.Errorf("feedback-service status %d", resp.StatusCode)
	}
	var out struct {
		PostID   uint64 `json:"post_id"`
		Likes    int64  `json:"likes"`
		Comments int64  `json:"comments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, 0, err
	}
	return out.Likes, out.Comments, nil
}
