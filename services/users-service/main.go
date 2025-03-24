package main

import (
	"fmt"
	"net/http"
	"users-service/configs"
	"users-service/pkg/di"
)

func main() {
	// Initialize config (e.g., environment variables, etc.)
	cfg := configs.NewConfig()

	// Build the DI container (services, repositories, etc.)
	container := di.NewContainer(cfg)

	// Register the user handlers
	http.HandleFunc("/users", container.UserHandler.HandleUsers)
	http.HandleFunc("/users/", container.UserHandler.HandleUserByID)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("Starting server on %s\n", addr)
	http.ListenAndServe(addr, nil)
}
