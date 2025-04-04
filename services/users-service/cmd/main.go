package main

import (
	"fmt"
	"net/http"
	"users-service/configs"
	"users-service/internal/auth"
	"users-service/internal/cities"
	"users-service/internal/follows"
	"users-service/internal/friends"
	"users-service/internal/interests"
	"users-service/internal/relationships"
	"users-service/internal/user"
	"users-service/internal/userdata"
	"users-service/pkg/db"
	"users-service/pkg/middleware"
)

func App() http.Handler {
	cfg := configs.LoadConfig()
	database := db.NewDb(cfg)

	database.DB.AutoMigrate(
		&user.User{},
		&userdata.UserData{},
		&cities.City{},
		&interests.Interest{},
		&interests.InterestUser{},
		&follows.Follow{},
		&friends.Friend{},
		&relationships.Relationship{},
	)

	router := http.NewServeMux()

	userRepo := user.NewUserRepository(database)
	authService := auth.NewAuthService(userRepo)

	userDataRepo := userdata.NewRepository(database.DB)
	userDataService := userdata.NewService(userDataRepo)

	cityRepo := cities.NewRepository(database.DB)
	cityService := cities.NewService(cityRepo)

	interestRepo := interests.NewRepository(database.DB)
	interestService := interests.NewService(interestRepo)

	followsRepo := follows.NewRepository(database.DB)
	followsService := follows.NewService(followsRepo)

	friendsRepo := friends.NewRepository(database.DB)
	friendsService := friends.NewService(friendsRepo)

	relationshipsRepo := relationships.NewRepository(database.DB)
	relationshipsService := relationships.NewService(relationshipsRepo)

	auth.NewAuthHandler(router, auth.AuthHandlerDeps{
		Config:      cfg,
		AuthService: authService,
	})

	userdata.NewHandler(router, userdata.HandlerDeps{
		Config:  cfg,
		Service: userDataService,
	})

	cities.NewHandler(router, cities.HandlerDeps{
		Config:  cfg,
		Service: cityService,
	})

	interests.NewHandler(router, interests.HandlerDeps{
		Config:  cfg,
		Service: interestService,
	})

	follows.NewHandler(router, follows.HandlerDeps{
		Config:  cfg,
		Service: followsService,
	})

	friends.NewHandler(router, friends.HandlerDeps{
		Config:  cfg,
		Service: friendsService,
	})

	relationships.NewHandler(router, relationships.HandlerDeps{
		Config:  cfg,
		Service: relationshipsService,
	})

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
