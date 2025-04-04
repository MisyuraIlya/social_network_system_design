package likes

import "errors"

type Service interface {
	LikePost(userID, postID, commentID uint) error
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) LikePost(userID, postID, commentID uint) error {
	if postID == 0 && commentID == 0 {
		return errors.New("no valid post or comment to like")
	}
	like := &Like{
		UserID:    userID,
		PostID:    postID,
		CommentID: commentID,
	}
	if err := s.repo.AddLike(like); err != nil {
		return err
	}
	if postID != 0 {
		if err := s.repo.IncrementPostLikes(postID); err != nil {
			return err
		}
	}
	return nil
}
