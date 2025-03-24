package user

import (
	"gorm.io/gorm"
)

type IUserRepository interface {
	Create(u *User) (*User, error)
	FindByEmail(email string) (*User, error)
	FindAll() ([]User, error)
	FindByID(id uint) (*User, error)
}

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) IUserRepository {
	return &UserRepository{DB: db}
}

func (repo *UserRepository) Create(u *User) (*User, error) {
	if err := repo.DB.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (repo *UserRepository) FindByEmail(email string) (*User, error) {
	var user User
	if err := repo.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (repo *UserRepository) FindAll() ([]User, error) {
	var users []User
	if err := repo.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (repo *UserRepository) FindByID(id uint) (*User, error) {
	var user User
	if err := repo.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
