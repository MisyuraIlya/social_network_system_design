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
	kafkaBroker := os.Getenv("KAFKA_BROKER")

	fmt.Printf("Feedback Gateway started. Kafka: %s\n", kafkaBroker)

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		pgHost, pgUser, pgPass, pgDB)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Could not connect to Postgres: %v", err)
	}
	defer db.Close()

	// Example create table for likes/comments
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS post_likes (
        id SERIAL PRIMARY KEY,
        post_id INT,
        user_id INT,
        created_at TIMESTAMP DEFAULT NOW()
    );`)
	if err != nil {
		log.Printf("Could not create post_likes table: %v\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS post_comments (
        id SERIAL PRIMARY KEY,
        post_id INT,
        user_id INT,
        comment TEXT,
        created_at TIMESTAMP DEFAULT NOW()
    );`)
	if err != nil {
		log.Printf("Could not create post_comments table: %v\n", err)
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Feedback Gateway is healthy"))
	})

	http.HandleFunc("/like", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Insert a like
			_, _ = w.Write([]byte("Post liked"))
			// Possibly publish to Kafka for feed updates
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/comment", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Insert a comment
			_, _ = w.Write([]byte("Comment added"))
			// Possibly publish to Kafka
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("Starting Feedback Gateway on port 8005...")
	log.Fatal(http.ListenAndServe(":8005", nil))
}
