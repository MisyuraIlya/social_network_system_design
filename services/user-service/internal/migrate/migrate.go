package migrate

import (
	"users-service/internal/interest"
	"users-service/internal/profile"
	"users-service/internal/shared/db"
	"users-service/internal/social"
	"users-service/internal/user"
)

func AutoMigrateAll(store *db.Store, shardID int) error {
	return store.Write(shardID).AutoMigrate(
		&user.User{},
		&profile.Profile{},
		&interest.City{}, &interest.Interest{}, &interest.InterestUser{},
		&social.Follow{}, &social.Friend{}, &social.Relationship{},
	)
}
