package cities

type Service interface {
	CreateCity(name string) (*City, error)
	ListCities() ([]City, error)
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) CreateCity(name string) (*City, error) {
	c := City{Name: name}
	err := s.repo.Create(&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *service) ListCities() ([]City, error) {
	return s.repo.List()
}
