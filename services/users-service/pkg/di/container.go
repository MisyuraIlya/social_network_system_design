package di

import (
	"users-service/configs"
	"users-service/internal/user"
	"users-service/pkg/db"
)

type Container struct {
	UserHandler *user.Handler
}

func NewContainer(cfg *configs.Config) *Container {
	db.ConnectDB()
	userRepo := user.NewInMemoryRepository()
	userService := user.NewService(userRepo)
	userHandler := &user.Handler{
		Service: userService,
	}

	return &Container{
		UserHandler: userHandler,
	}
}
