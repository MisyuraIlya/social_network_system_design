package likes

type Like struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	PostID    uint
	CommentID uint
}
