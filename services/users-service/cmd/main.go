package main

import (
	"fmt"
	"net/http"

	"users-service/configs"
	"users-service/internal/auth"
	"users-service/internal/user"
	"users-service/pkg/db"
	"users-service/pkg/middleware"
)

func App() http.Handler {
	cfg := configs.LoadConfig()
	database := db.NewDb(cfg)
	database.DB.AutoMigrate(&user.User{})

	router := http.NewServeMux()

	// repositories
	userRepository := user.NewUserRepository(database)

	// services
	authService := auth.NewAuthService(userRepository)

	// handlers
	auth.NewAuthHandler(router, auth.AuthHandlerDeps{
		Config:      cfg,
		AuthService: authService,
	})

	// middlewares
	stack := middleware.Chain(
		middleware.CORS,
		middleware.Logging,
	)
	return stack(router)
}

func main() {
	app := App()
	server := http.Server{
		Addr:    ":8081",
		Handler: app,
	}
	fmt.Println("User Service listening on port 8081")
	server.ListenAndServe()
}
