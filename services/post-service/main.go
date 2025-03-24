package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq" // Postgres driver
)

func main() {
	pgHost := os.Getenv("POSTGRES_HOST")
	pgUser := os.Getenv("POSTGRES_USER")
	pgPass := os.Getenv("POSTGRES_PASSWORD")
	pgDB := os.Getenv("POSTGRES_DB")

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		pgHost, pgUser, pgPass, pgDB)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Could not connect to Postgres: %v", err)
	}
	defer db.Close()

	// Basic table creation (for example)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS posts (
        id SERIAL PRIMARY KEY,
        user_id INT NOT NULL,
        text TEXT,
        created_at TIMESTAMP DEFAULT NOW()
    );`)
	if err != nil {
		log.Printf("Could not create table: %v\n", err)
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Post Service is healthy"))
	})

	http.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			// Parse request & insert post into DB
			_, _ = w.Write([]byte("Create a new post"))
		case http.MethodGet:
			// Retrieve posts from DB
			_, _ = w.Write([]byte("Get all posts or by user_id"))
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("Starting Post Service on port 8001...")
	log.Fatal(http.ListenAndServe(":8001", nil))
}
