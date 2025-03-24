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

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		pgHost, pgUser, pgPass, pgDB)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Could not connect to Postgres: %v", err)
	}
	defer db.Close()

	// Possibly create a table for relationships
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS relations (
        id SERIAL PRIMARY KEY,
        user_id INT NOT NULL,
        friend_id INT NOT NULL,
        created_at TIMESTAMP DEFAULT NOW()
    );`)
	if err != nil {
		log.Printf("Could not create relations table: %v\n", err)
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Relation Service is healthy"))
	})

	http.HandleFunc("/follow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Insert new relation record
			_, _ = w.Write([]byte("Follow request accepted"))
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("Starting Relation Service on port 8004...")
	log.Fatal(http.ListenAndServe(":8004", nil))
}
