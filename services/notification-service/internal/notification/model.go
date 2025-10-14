package notification

import "time"

type Kind string

const (
	KindMessage Kind = "message"
	KindPost    Kind = "post"
)

type Notification struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Kind      Kind           `json:"kind"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Meta      map[string]any `json:"meta,omitempty"`
	Read      bool           `json:"read"`
	CreatedAt time.Time      `json:"created_at"`
}
