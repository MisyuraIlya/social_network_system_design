package feedback

type IFeedbackService interface {
	LikePost(userID, postID uint) (*Like, error)
	UnlikePost(userID, postID uint) error
	CountLikes(postID uint) (int64, error)

	AddComment(userID, postID uint, content string) (*Comment, error)
	DeleteComment(commentID uint) error
	GetComments(postID uint) ([]Comment, error)
}

type FeedbackService struct {
	repo IFeedbackRepository
}

func NewFeedbackService(r IFeedbackRepository) IFeedbackService {
	return &FeedbackService{repo: r}
}

func (s *FeedbackService) LikePost(userID, postID uint) (*Like, error) {
	like := &Like{UserID: userID, PostID: postID}
	return s.repo.CreateLike(like)
}

func (s *FeedbackService) UnlikePost(userID, postID uint) error {
	return s.repo.DeleteLike(userID, postID)
}

func (s *FeedbackService) CountLikes(postID uint) (int64, error) {
	return s.repo.CountLikes(postID)
}

func (s *FeedbackService) AddComment(userID, postID uint, content string) (*Comment, error) {
	c := &Comment{
		UserID:  userID,
		PostID:  postID,
		Content: content,
	}
	return s.repo.CreateComment(c)
}

func (s *FeedbackService) DeleteComment(commentID uint) error {
	return s.repo.DeleteComment(commentID)
}

func (s *FeedbackService) GetComments(postID uint) ([]Comment, error) {
	return s.repo.FindCommentsByPost(postID)
}
