package like

type Service interface {
	Like(uid string, postID uint64) (int64, error)
	Unlike(uid string, postID uint64) (int64, error)
	Get(postID uint64, uid string) (int64, bool, error)
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Like(uid string, postID uint64) (int64, error)   { return s.repo.Like(uid, postID) }
func (s *service) Unlike(uid string, postID uint64) (int64, error) { return s.repo.Unlike(uid, postID) }
func (s *service) Get(postID uint64, uid string) (int64, bool, error) {
	return s.repo.GetCount(postID, uid)
}
