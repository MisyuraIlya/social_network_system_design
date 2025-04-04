package relationships

import "time"

type Service interface {
	CreateRelationship(userID, relatedID, relType int) error
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) CreateRelationship(userID, relatedID, relType int) error {
	rel := Relationship{
		UserID:           userID,
		RelatedID:        relatedID,
		RelationshipType: relType,
		CreatedAt:        time.Now(),
	}
	return s.repo.Create(&rel)
}
