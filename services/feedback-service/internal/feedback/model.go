package feedback

type Like struct {
	ID     uint   `gorm:"primaryKey"`
	UserID string `gorm:"not null"`
	PostID string `gorm:"not null"`
}

type Comment struct {
	ID      uint   `gorm:"primaryKey"`
	UserID  string `gorm:"not null"`
	PostID  string `gorm:"not null"`
	Content string `gorm:"not null"`
}
