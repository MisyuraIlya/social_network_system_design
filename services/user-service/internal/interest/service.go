package interest

type Service interface {
	Create(shardID int, name string) (*Interest, error)

	Attach(uid string, interestID uint64) error
	Detach(uid string, interestID uint64) error
	List(uid string, limit, offset int) ([]Interest, error)
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Create(shardID int, name string) (*Interest, error) {
	return s.repo.Create(shardID, name)
}
func (s *service) Attach(uid string, interestID uint64) error { return s.repo.Attach(uid, interestID) }
func (s *service) Detach(uid string, interestID uint64) error { return s.repo.Detach(uid, interestID) }
func (s *service) List(uid string, limit, offset int) ([]Interest, error) {
	return s.repo.List(uid, limit, offset)
}
