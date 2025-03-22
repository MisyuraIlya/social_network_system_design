package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
	minioSecretKey := os.Getenv("MINIO_SECRET_KEY")

	fmt.Printf("Media Service config:\n")
	fmt.Printf("MinIO Endpoint: %s\n", minioEndpoint)
	fmt.Printf("MinIO Access Key: %s\n", minioAccessKey)
	fmt.Printf("MinIO Secret Key: %s\n", minioSecretKey)

	// Here you might initialize a MinIO or S3 client using the AWS SDK or MinIO SDK

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Media Service is healthy"))
	})

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Parse the uploaded file and store in MinIO
		_, _ = w.Write([]byte("Media file uploaded to MinIO (mock)"))
	})

	log.Println("Starting Media Service on port 8002...")
	log.Fatal(http.ListenAndServe(":8002", nil))
}
