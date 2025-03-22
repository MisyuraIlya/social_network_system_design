package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	pgHost := os.Getenv("POSTGRES_HOST")
	pgUser := os.Getenv("POSTGRES_USER")
	pgPass := os.Getenv("POSTGRES_PASSWORD")
	pgDB := os.Getenv("POSTGRES_DB")

	// Could also connect to Kafka here, create consumer to read post events
	kafkaBroker := os.Getenv("KAFKA_BROKER")

	fmt.Printf("Feed Service started with Kafka: %s\n", kafkaBroker)

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		pgHost, pgUser, pgPass, pgDB)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Could not connect to Postgres: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Feed Service is healthy"))
	})

	http.HandleFunc("/feed", func(w http.ResponseWriter, r *http.Request) {
		// Retrieve user feed from DB or in-memory store
		_, _ = w.Write([]byte("Return user feed here"))
	})

	log.Println("Starting Feed Service on port 8003...")
	log.Fatal(http.ListenAndServe(":8003", nil))
}
