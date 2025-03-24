package feed

import "gorm.io/gorm"

type IFeedRepository interface {
	Create(item *FeedItem) (*FeedItem, error)
	FindByUser(userID uint) ([]FeedItem, error)
}

type FeedRepository struct {
	DB *gorm.DB
}

func NewFeedRepository(db *gorm.DB) IFeedRepository {
	return &FeedRepository{DB: db}
}

func (repo *FeedRepository) Create(item *FeedItem) (*FeedItem, error) {
	if err := repo.DB.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (repo *FeedRepository) FindByUser(userID uint) ([]FeedItem, error) {
	var items []FeedItem
	if err := repo.DB.Where("user_id = ?", userID).
		Order("id desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
