package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Read environment variables for DB, Kafka, etc.
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	pgHost := os.Getenv("POSTGRES_HOST")
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")

	// For demonstration, just printing out the config
	fmt.Printf("API Gateway started with config:\n")
	fmt.Printf("Kafka Broker: %s\n", kafkaBroker)
	fmt.Printf("Postgres Host: %s\n", pgHost)
	fmt.Printf("MinIO Endpoint: %s\n", minioEndpoint)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("API Gateway is healthy!"))
	})

	// Example routes that would proxy or orchestrate calls to microservices:
	http.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		// Here you might forward request to Post Service
		_, _ = w.Write([]byte("List or Create Posts (proxy to Post Service)"))
	})

	http.HandleFunc("/media/upload", func(w http.ResponseWriter, r *http.Request) {
		// Here you might forward request to Media Service
		_, _ = w.Write([]byte("Upload media (proxy to Media Service)"))
	})

	http.HandleFunc("/feed", func(w http.ResponseWriter, r *http.Request) {
		// Here you might forward request to Feed Service
		_, _ = w.Write([]byte("Get user feed (proxy to Feed Service)"))
	})

	log.Println("Starting API Gateway on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
