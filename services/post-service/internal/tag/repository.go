package tag

import (
	"post-service/internal/shared/db"

	"gorm.io/gorm"
)

type Repository interface {
	FirstOrCreateByName(name string) (*Tag, error)
	FindByNames(names []string) ([]Tag, error)
}

type repo struct{ store *db.Store }

func NewRepository(s *db.Store) Repository { return &repo{store: s} }

func (r *repo) FirstOrCreateByName(name string) (*Tag, error) {
	t := &Tag{Name: name}
	if err := r.store.Base.FirstOrCreate(t, "name = ?", name).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func (r *repo) FindByNames(names []string) ([]Tag, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var out []Tag
	err := r.store.Base.Where("name IN ?", names).Find(&out).Error
	return out, err
}

var _ = gorm.ErrRecordNotFound
