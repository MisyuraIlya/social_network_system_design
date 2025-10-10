package user

import (
	"errors"
	"log"

	"users-service/pkg/db"
	"users-service/pkg/shard"

	"gorm.io/gorm"
)

type ShardPicker interface {
	Pick(shardID int) *gorm.DB
	ForcePrimary(shardID int) *gorm.DB
	// For logging (optional)
	ShardInfo(shardID int) (db.ShardCfg, bool)
}

type IUserRepository interface {
	Create(u *User) (*User, error)
	FindByEmail(email string, shardID int) (*User, error)
	FindByUserID(uid string) (*User, error)
	FindAllByShard(shardID int) ([]User, error)
	FindAllByShardPaged(shardID, limit, offset int) ([]User, error)
}

type UserRepository struct {
	db ShardPicker
}

func NewUserRepository(p ShardPicker) IUserRepository {
	return &UserRepository{db: p}
}

func (r *UserRepository) logShard(where, role string, shardID int) {
	if cfg, ok := r.db.ShardInfo(shardID); ok {
		w := db.RedactDSN(cfg.Writer)
		var readers string
		if len(cfg.Readers) > 0 {
			readers = db.RedactDSN(cfg.Readers[0])
			if len(cfg.Readers) > 1 {
				readers += " (+more)"
			}
		}
		log.Printf("[repo:%s] role=%s shard=%d writer=[%s] reader0=[%s]", where, role, shardID, w, readers)
	} else {
		log.Printf("[repo:%s] role=%s shard=%d", where, role, shardID)
	}
}

func (r *UserRepository) Create(u *User) (*User, error) {
	r.logShard("Create", "primary", u.ShardID)
	if err := r.db.ForcePrimary(u.ShardID).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(email string, shardID int) (*User, error) {
	r.logShard("FindByEmail", "replica", shardID)
	var u User
	if err := r.db.Pick(shardID).Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByUserID(uid string) (*User, error) {
	sh, ok := shard.ExtractShard(uid)
	if !ok {
		return nil, errors.New("invalid user_id format")
	}
	r.logShard("FindByUserID", "replica", sh)
	var u User
	if err := r.db.Pick(sh).Where("user_id = ?", uid).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindAllByShard(shardID int) ([]User, error) {
	r.logShard("FindAllByShard", "replica", shardID)
	var users []User
	if err := r.db.Pick(shardID).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) FindAllByShardPaged(shardID, limit, offset int) ([]User, error) {
	r.logShard("FindAllByShardPaged", "replica", shardID)
	var users []User
	if err := r.db.Pick(shardID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
