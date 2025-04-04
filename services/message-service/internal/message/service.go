package message

type Service interface {
	CreateMessage(userID uint, content string) (*Message, error)
	ListMessages() ([]Message, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateMessage(userID uint, content string) (*Message, error) {
	msg := &Message{
		UserID:  userID,
		Content: content,
	}
	err := s.repo.Save(msg)
	return msg, err
}

func (s *service) ListMessages() ([]Message, error) {
	return s.repo.FindAll()
}
