package interest

import (
	"users-service/internal/shared/db"
	"users-service/internal/shared/shard"

	"gorm.io/gorm"
)

type Repository interface {
	// NEW:
	Create(shardID int, name string) (*Interest, error)

	Attach(uid string, interestID uint64) error
	Detach(uid string, interestID uint64) error
	List(uid string, limit, offset int) ([]Interest, error)
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Create(shardID int, name string) (*Interest, error) {
	in := &Interest{Name: name}
	if err := r.store.Write(shardID).FirstOrCreate(in, "name = ?", name).Error; err != nil {
		return nil, err
	}
	return in, nil
}

func (r *repo) Attach(uid string, interestID uint64) error {
	sh, _ := shard.Extract(uid)
	return r.store.Write(sh).FirstOrCreate(&InterestUser{UserID: uid, InterestID: interestID}).Error
}
func (r *repo) Detach(uid string, interestID uint64) error {
	sh, _ := shard.Extract(uid)
	return r.store.Write(sh).Delete(&InterestUser{}, "user_id=? AND interest_id=?", uid, interestID).Error
}
func (r *repo) List(uid string, limit, offset int) ([]Interest, error) {
	sh, _ := shard.Extract(uid)
	var ints []Interest
	err := r.store.Use(sh).
		Joins("JOIN interest_users iu ON iu.interest_id = interests.id AND iu.user_id = ?", uid).
		Model(&Interest{}).Limit(limit).Offset(offset).Find(&ints).Error
	return ints, err
}

var _ = gorm.ErrRecordNotFound
