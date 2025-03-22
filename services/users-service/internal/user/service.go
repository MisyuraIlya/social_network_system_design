package user

type Service interface {
	CreateUser(payload UserCreatePayload) (User, error)
	GetAllUsers() ([]User, error)
	GetUserByID(id int) (User, error)
	UpdateUser(id int, payload UserUpdatePayload) (User, error)
	DeleteUser(id int) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateUser(payload UserCreatePayload) (User, error) {
	user := User{
		Name:  payload.Name,
		Email: payload.Email,
	}
	return s.repo.Create(user)
}

func (s *service) GetAllUsers() ([]User, error) {
	return s.repo.GetAll()
}

func (s *service) GetUserByID(id int) (User, error) {
	return s.repo.GetByID(id)
}

func (s *service) UpdateUser(id int, payload UserUpdatePayload) (User, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return User{}, err
	}
	if payload.Name != "" {
		existing.Name = payload.Name
	}
	if payload.Email != "" {
		existing.Email = payload.Email
	}
	return s.repo.Update(existing)
}

func (s *service) DeleteUser(id int) error {
	return s.repo.Delete(id)
}
