package main

import (
	"fmt"
	"log"
	"net/http"

	"users-service/internal/user"
	multidb "users-service/pkg/db"

	"gorm.io/gorm"
)

func main() {
	// Open all shard connections from SHARDS_JSON
	mdb := multidb.OpenMultiFromEnv()

	// Auto-migrate the users table on every shard
	if err := mdb.Range(func(id int, db *gorm.DB) error {
		return db.AutoMigrate(&user.User{})
	}); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Wire repository/service/handler
	repo := user.NewUserRepository(mdb) // multi-shard aware
	svc := user.NewUserService(repo)    // picks shard per request
	handler := user.NewUserHandler(svc) // HTTP

	mux := http.NewServeMux()
	user.RegisterRoutes(mux, handler)

	addr := ":8081"
	fmt.Printf("User Service listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
