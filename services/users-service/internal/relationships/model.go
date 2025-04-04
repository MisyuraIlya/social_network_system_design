package relationships

import "time"

type Relationship struct {
	UserID           int
	RelatedID        int
	RelationshipType int
	CreatedAt        time.Time
}
