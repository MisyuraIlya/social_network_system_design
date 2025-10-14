package comment

type Service interface {
	Create(uid string, postID uint64, in CreateReq) (*PostComment, error)
	DeleteMine(uid string, commentID uint64) error
	ListByPost(postID uint64, limit, offset int) ([]PostComment, error)
	CommentCount(postID uint64) (int64, error)
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Create(uid string, postID uint64, in CreateReq) (*PostComment, error) {
	return s.repo.Create(uid, postID, in)
}
func (s *service) DeleteMine(uid string, commentID uint64) error {
	return s.repo.DeleteMine(uid, commentID)
}
func (s *service) ListByPost(postID uint64, limit, offset int) ([]PostComment, error) {
	return s.repo.ListByPost(postID, limit, offset)
}
func (s *service) CommentCount(postID uint64) (int64, error) {
	_, c, err := s.repo.Counts(postID)
	return c, err
}
