package feedback

import "gorm.io/gorm"

type IFeedbackRepository interface {
	CreateLike(like *Like) (*Like, error)
	DeleteLike(userID, postID uint) error
	CountLikes(postID uint) (int64, error)

	CreateComment(comment *Comment) (*Comment, error)
	DeleteComment(commentID uint) error
	FindCommentsByPost(postID uint) ([]Comment, error)
}

type FeedbackRepository struct {
	DB *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) IFeedbackRepository {
	return &FeedbackRepository{DB: db}
}

func (repo *FeedbackRepository) CreateLike(like *Like) (*Like, error) {
	if err := repo.DB.Create(like).Error; err != nil {
		return nil, err
	}
	return like, nil
}

func (repo *FeedbackRepository) DeleteLike(userID, postID uint) error {
	return repo.DB.Where("user_id = ? AND post_id = ?", userID, postID).Delete(&Like{}).Error
}

func (repo *FeedbackRepository) CountLikes(postID uint) (int64, error) {
	var count int64
	err := repo.DB.Model(&Like{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

func (repo *FeedbackRepository) CreateComment(comment *Comment) (*Comment, error) {
	if err := repo.DB.Create(comment).Error; err != nil {
		return nil, err
	}
	return comment, nil
}

func (repo *FeedbackRepository) DeleteComment(commentID uint) error {
	return repo.DB.Delete(&Comment{}, commentID).Error
}

func (repo *FeedbackRepository) FindCommentsByPost(postID uint) ([]Comment, error) {
	var comments []Comment
	if err := repo.DB.Where("post_id = ?", postID).Order("id asc").Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}
