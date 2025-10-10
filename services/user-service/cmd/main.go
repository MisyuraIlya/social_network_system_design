package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"users-service/internal/user"
	"users-service/pkg/db"

	"gorm.io/gorm"
)

type ShardPicker interface {
	Pick(shardID int) *gorm.DB
	ForcePrimary(shardID int) *gorm.DB
}

func main() {
	store := db.OpenFromEnv()

	if os.Getenv("AUTO_MIGRATE") == "true" {
		numShards := mustAtoi(os.Getenv("NUM_SHARDS"))
		for i := 0; i < numShards; i++ {
			if err := store.ForcePrimary(i).AutoMigrate(&user.User{}); err != nil {
				log.Fatalf("migration failed on shard %d: %v", i, err)
			}
		}
	}

	repo := user.NewUserRepository(store)
	svc := user.NewUserService(repo)
	handler := user.NewUserHandler(svc)

	mux := http.NewServeMux()
	user.RegisterRoutes(mux, handler)

	addr := os.Getenv("APP_PORT")
	if addr == "" {
		addr = ":8081"
	}
	fmt.Printf("User Service listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		n = 1
	}
	return n
}
