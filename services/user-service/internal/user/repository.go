package user

import (
	"errors"

	"users-service/pkg/shard"

	"gorm.io/gorm"
)

type ShardedDB interface {
	Get(shardID int) *gorm.DB
}

type IUserRepository interface {
	Create(u *User) (*User, error)
	FindByEmail(email string, shardID int) (*User, error)
	FindByUserID(uid string) (*User, error)
	FindAllByShard(shardID int) ([]User, error)
	FindAllByShardPaged(shardID, limit, offset int) ([]User, error)
}

type UserRepository struct {
	mdb ShardedDB
}

func NewUserRepository(mdb ShardedDB) IUserRepository {
	return &UserRepository{mdb: mdb}
}

func (r *UserRepository) Create(u *User) (*User, error) {
	if err := r.mdb.Get(u.ShardID).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(email string, shardID int) (*User, error) {
	var u User
	if err := r.mdb.Get(shardID).Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByUserID(uid string) (*User, error) {
	sh, ok := shard.ExtractShard(uid)
	if !ok {
		return nil, errors.New("invalid user_id format")
	}
	var u User
	if err := r.mdb.Get(sh).Where("user_id = ?", uid).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindAllByShard(shardID int) ([]User, error) {
	var users []User
	if err := r.mdb.Get(shardID).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) FindAllByShardPaged(shardID, limit, offset int) ([]User, error) {
	var users []User
	if err := r.mdb.Get(shardID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
