package social

import "time"

type Follow struct {
	UserID    string `gorm:"primaryKey;size:64"`
	TargetID  string `gorm:"primaryKey;size:64"`
	CreatedAt time.Time
}
type Friend struct {
	UserID    string `gorm:"primaryKey;size:64"`
	FriendID  string `gorm:"primaryKey;size:64"`
	CreatedAt time.Time
}
type Relationship struct {
	UserID    string `gorm:"primaryKey;size:64"`
	RelatedID string `gorm:"primaryKey;size:64"`
	Type      int    `gorm:"primaryKey"`
	CreatedAt time.Time
}
