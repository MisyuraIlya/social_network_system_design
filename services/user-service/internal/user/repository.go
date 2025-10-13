package user

import (
	"errors"
	"users-service/internal/shared/db"
	"users-service/internal/shared/shard"
)

type Repository interface {
	Create(u *User) (*User, error)
	GetByEmail(email string, shardID int) (*User, error)
	GetByUserID(uid string) (*User, error)
	ListByShard(shardID, limit, offset int) ([]User, error)
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(u *User) (*User, error) {
	if err := r.store.Write(u.ShardID).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}
func (r *repo) GetByEmail(email string, shardID int) (*User, error) {
	var u User
	err := r.store.Use(shardID).Where("email = ?", email).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}
func (r *repo) GetByUserID(uid string) (*User, error) {
	sh, ok := shard.Extract(uid)
	if !ok {
		return nil, errors.New("bad user_id")
	}
	var u User
	if err := r.store.Use(sh).Where("user_id = ?", uid).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
func (r *repo) ListByShard(shardID, limit, offset int) ([]User, error) {
	var out []User
	err := r.store.Use(shardID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error
	return out, err
}
