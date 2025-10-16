package migrate

import (
	"message-service/internal/chat"
	"message-service/internal/message"
	"message-service/internal/shared/db"
)

func AutoMigrateAll(store *db.Store) error {
	return store.Base.AutoMigrate(
		&chat.Chat{}, &chat.ChatUser{},
		&message.Message{},
		&message.MessageSeen{},
	)
}
