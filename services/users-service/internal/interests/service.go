package interests

type Service interface {
	CreateInterest(name string) (*Interest, error)
	ListInterests() ([]Interest, error)
	AddUserInterest(userID, interestID int) error
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) CreateInterest(name string) (*Interest, error) {
	i := Interest{Name: name}
	err := s.repo.CreateInterest(&i)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (s *service) ListInterests() ([]Interest, error) {
	return s.repo.ListInterests()
}

func (s *service) AddUserInterest(userID, interestID int) error {
	iu := InterestUser{
		UserID:     userID,
		InterestID: interestID,
	}
	return s.repo.AddUserInterest(&iu)
}
