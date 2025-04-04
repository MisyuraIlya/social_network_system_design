package media

import (
	"context"
	"log"

	"media-service/configs"

	"github.com/minio/minio-go/v7"
)

func InitializeHandler(cfg *configs.Config) *Handler {
	repo, err := NewS3Repository(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey)
	if err != nil {
		log.Fatalf("Failed to initialize S3 repository: %v", err)
	}

	ctx := context.Background()

	// Ensure bucket is created (idempotent call).
	makeBucketOpts := minio.MakeBucketOptions{}
	err = repo.client.MakeBucket(ctx, cfg.S3BucketName, makeBucketOpts)
	if err != nil {
		exists, errBucketExists := repo.client.BucketExists(ctx, cfg.S3BucketName)
		if errBucketExists != nil || !exists {
			log.Fatalf("Failed to create/check bucket: %v", err)
		}
	}

	service := NewMediaService(repo, cfg.S3BucketName)
	handler := NewHandler(service)
	return handler
}
