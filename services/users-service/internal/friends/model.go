package friends

import "time"

type Friend struct {
	UserID    int
	FriendID  int
	CreatedAt time.Time
}
