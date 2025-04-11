package friends

import "gorm.io/gorm"

type Repository interface {
	CreateFriend(f *Friend) error
	GetFriends(userID int) ([]Friend, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateFriend(f *Friend) error {
	return r.db.Create(f).Error
}

func (r *repository) GetFriends(userID int) ([]Friend, error) {
	var friends []Friend
	if err := r.db.Where("user_id = ?", userID).Find(&friends).Error; err != nil {
		return nil, err
	}
	return friends, nil
}
