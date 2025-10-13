package profile

import (
	"users-service/internal/shared/db"
	"users-service/internal/shared/shard"

	"gorm.io/gorm/clause"
)

type Repository interface {
	Upsert(p *Profile) error
	GetPublic(userID string) (*Profile, error)
}
type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) Upsert(p *Profile) error {
	sh, _ := shard.Extract(p.UserID)
	return r.store.Write(sh).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"description", "city_id", "education", "hobby", "updated_at"}),
	}).Create(p).Error
}
func (r *repo) GetPublic(uid string) (*Profile, error) {
	sh, _ := shard.Extract(uid)
	var p Profile
	if err := r.store.Use(sh).First(&p, "user_id = ?", uid).Error; err != nil {
		return nil, err
	}
	return &p, nil
}
