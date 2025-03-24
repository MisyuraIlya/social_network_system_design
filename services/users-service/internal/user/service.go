package user

import "errors"

type IUserService interface {
	Register(email, password, name string) (*User, error)
	Login(email, password string) (*User, error)
	ListAll() ([]User, error)
	GetByID(id uint) (*User, error)
}

type UserService struct {
	repo IUserRepository
}

func NewUserService(repo IUserRepository) IUserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(email, password, name string) (*User, error) {
	existing, _ := s.repo.FindByEmail(email)
	if existing != nil {
		return nil, errors.New("user already exists")
	}
	newUser := &User{
		Email:    email,
		Password: password,
		Name:     name,
	}
	return s.repo.Create(newUser)
}

func (s *UserService) Login(email, password string) (*User, error) {
	usr, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("wrong credentials")
	}
	if usr.Password != password {
		return nil, errors.New("wrong password")
	}
	return usr, nil
}

func (s *UserService) ListAll() ([]User, error) {
	return s.repo.FindAll()
}

func (s *UserService) GetByID(id uint) (*User, error) {
	return s.repo.FindByID(id)
}
