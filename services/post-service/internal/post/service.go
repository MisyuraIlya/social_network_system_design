package post

type IPostService interface {
	CreatePost(userID uint, content string) (*Post, error)
	GetPost(id uint) (*Post, error)
	ListPosts() ([]Post, error)
}

type PostService struct {
	repo IPostRepository
}

func NewPostService(r IPostRepository) IPostService {
	return &PostService{repo: r}
}

func (s *PostService) CreatePost(userID uint, content string) (*Post, error) {
	post := &Post{
		UserID:  userID,
		Content: content,
	}
	return s.repo.Create(post)
}

func (s *PostService) GetPost(id uint) (*Post, error) {
	return s.repo.FindByID(id)
}

func (s *PostService) ListPosts() ([]Post, error) {
	return s.repo.ListAll()
}
