package main

import (
	"fmt"
	"log"
	"net/http"

	"users-service/configs"
	"users-service/internal/user"
	"users-service/pkg/db"
)

func main() {
	cfg := configs.LoadConfig()

	database := db.NewDb(cfg)
	database.DB.AutoMigrate(&user.User{})

	repo := user.NewUserRepository(database.DB)
	svc := user.NewUserService(repo)

	handler := user.NewUserHandler(svc)

	mux := http.NewServeMux()
	user.RegisterRoutes(mux, handler)

	srv := &http.Server{
		Addr:    cfg.AppPort,
		Handler: mux,
	}

	fmt.Printf("User Service listening on %s\n", cfg.AppPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
