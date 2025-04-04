package tags

type Service interface {
	CreateTag(name string) error
	GetAllTags() ([]Tag, error)
	CreatePostTag(postID, tagID uint) error
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) CreateTag(name string) error {
	t := &Tag{
		Name: name,
	}
	return s.repo.CreateTag(t)
}

func (s *service) GetAllTags() ([]Tag, error) {
	return s.repo.GetAllTags()
}

func (s *service) CreatePostTag(postID, tagID uint) error {
	return s.repo.CreatePostTag(postID, tagID)
}
