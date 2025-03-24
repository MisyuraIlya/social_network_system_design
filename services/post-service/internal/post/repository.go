package post

import "gorm.io/gorm"

type IPostRepository interface {
	Create(p *Post) (*Post, error)
	FindByID(id uint) (*Post, error)
	ListAll() ([]Post, error)
}

type PostRepository struct {
	DB *gorm.DB
}

func NewPostRepository(db *gorm.DB) IPostRepository {
	return &PostRepository{DB: db}
}

func (repo *PostRepository) Create(p *Post) (*Post, error) {
	if err := repo.DB.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (repo *PostRepository) FindByID(id uint) (*Post, error) {
	var post Post
	if err := repo.DB.First(&post, id).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (repo *PostRepository) ListAll() ([]Post, error) {
	var posts []Post
	if err := repo.DB.Order("id desc").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}
