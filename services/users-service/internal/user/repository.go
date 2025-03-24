package user

import "errors"

type Repository interface {
	Create(u User) (User, error)
	GetAll() ([]User, error)
	GetByID(id int) (User, error)
	Update(u User) (User, error)
	Delete(id int) error
}

type InMemoryRepository struct {
	users  map[int]User
	nextID int
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		users:  make(map[int]User),
		nextID: 1,
	}
}

func (r *InMemoryRepository) Create(u User) (User, error) {
	u.ID = r.nextID
	r.users[r.nextID] = u
	r.nextID++
	return u, nil
}

func (r *InMemoryRepository) GetAll() ([]User, error) {
	all := make([]User, 0, len(r.users))
	for _, user := range r.users {
		all = append(all, user)
	}
	return all, nil
}

func (r *InMemoryRepository) GetByID(id int) (User, error) {
	u, ok := r.users[id]
	if !ok {
		return User{}, errors.New("user not found")
	}
	return u, nil
}

func (r *InMemoryRepository) Update(u User) (User, error) {
	if _, ok := r.users[u.ID]; !ok {
		return User{}, errors.New("user not found")
	}
	r.users[u.ID] = u
	return u, nil
}

func (r *InMemoryRepository) Delete(id int) error {
	if _, ok := r.users[id]; !ok {
		return errors.New("user not found")
	}
	delete(r.users, id)
	return nil
}
