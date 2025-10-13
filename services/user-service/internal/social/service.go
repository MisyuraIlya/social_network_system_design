package social

type Service interface {
	Follow(uid, target string) error
	Unfollow(uid, target string) error
	ListFollowing(uid string, limit, offset int) ([]string, error)
	Befriend(a, b string) error
	Unfriend(a, b string) error
	ListFriends(uid string, limit, offset int) ([]string, error)
	CreateRelationship(uid, related string, typ int) error
	DeleteRelationship(uid, related string, typ int) error
	ListRelationships(uid string, typ, limit, offset int) ([]string, error)
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Follow(uid, target string) error   { return s.repo.Follow(uid, target) }
func (s *service) Unfollow(uid, target string) error { return s.repo.Unfollow(uid, target) }
func (s *service) ListFollowing(uid string, limit, offset int) ([]string, error) {
	return s.repo.ListFollowing(uid, limit, offset)
}
func (s *service) Befriend(a, b string) error { return s.repo.Befriend(a, b) }
func (s *service) Unfriend(a, b string) error { return s.repo.Unfriend(a, b) }
func (s *service) ListFriends(uid string, limit, offset int) ([]string, error) {
	return s.repo.ListFriends(uid, limit, offset)
}
func (s *service) CreateRelationship(uid, related string, typ int) error {
	return s.repo.CreateRelationship(uid, related, typ)
}
func (s *service) DeleteRelationship(uid, related string, typ int) error {
	return s.repo.DeleteRelationship(uid, related, typ)
}
func (s *service) ListRelationships(uid string, typ, limit, offset int) ([]string, error) {
	return s.repo.ListRelationships(uid, typ, limit, offset)
}
