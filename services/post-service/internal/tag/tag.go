package tag

import "time"

type Tag struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:120" json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
