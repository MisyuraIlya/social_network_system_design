package userdata

type Service interface {
	GetUserData(userID int) (*UserData, error)
	UpdateUserData(ud *UserData) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetUserData(userID int) (*UserData, error) {
	return s.repo.GetByUserID(userID)
}

func (s *service) UpdateUserData(ud *UserData) error {
	return s.repo.CreateOrUpdate(ud)
}
