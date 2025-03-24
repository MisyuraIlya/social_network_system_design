package feed

type IFeedService interface {
	AddToFeed(userID, postID uint) (*FeedItem, error)
	GetFeedForUser(userID uint) ([]FeedItem, error)
}

type FeedService struct {
	repo IFeedRepository
}

func NewFeedService(r IFeedRepository) IFeedService {
	return &FeedService{repo: r}
}

func (s *FeedService) AddToFeed(userID, postID uint) (*FeedItem, error) {
	item := &FeedItem{
		UserID: userID,
		PostID: postID,
	}
	return s.repo.Create(item)
}

func (s *FeedService) GetFeedForUser(userID uint) ([]FeedItem, error) {
	return s.repo.FindByUser(userID)
}
