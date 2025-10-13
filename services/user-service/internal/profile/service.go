package profile

import "time"

type Service interface {
	Upsert(uid string, in UpsertReq) error
	GetPublic(uid string) (*Profile, error)
}
type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Upsert(uid string, in UpsertReq) error {
	return s.repo.Upsert(&Profile{
		UserID: uid, Description: in.Description, CityID: in.CityID,
		Education: in.Education, Hobby: in.Hobby, UpdatedAt: time.Now(),
	})
}
func (s *service) GetPublic(uid string) (*Profile, error) { return s.repo.GetPublic(uid) }
