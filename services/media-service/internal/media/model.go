package media

import "time"

type Media struct {
	ID          uint   `gorm:"primaryKey"`
	FileName    string `gorm:"size:255"`
	ContentType string `gorm:"size:100"`
	Size        int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
