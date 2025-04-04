package follows

import "time"

type Follow struct {
	UserID     int
	FollowedID int
	CreatedAt  time.Time
}
