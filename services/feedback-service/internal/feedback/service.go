package feedback

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Like(req LikeRequest) error {
	return s.repo.SaveLike(Like{
		UserID: req.UserID,
		PostID: req.PostID,
	})
}

func (s *Service) Comment(req CommentRequest) error {
	return s.repo.SaveComment(Comment{
		UserID:  req.UserID,
		PostID:  req.PostID,
		Content: req.Content,
	})
}
