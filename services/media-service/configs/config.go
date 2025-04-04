package configs

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort      string
	S3Endpoint   string
	S3AccessKey  string
	S3SecretKey  string
	S3BucketName string
}

func LoadConfig() (*Config, error) {
	// Replace defaults as needed:
	cfg := &Config{
		AppPort:      getEnv("MEDIA_APP_PORT", ":8084"),
		S3Endpoint:   getEnv("S3_ENDPOINT", "http://localhost:9000"),
		S3AccessKey:  getEnv("S3_ACCESS_KEY", "minio"),
		S3SecretKey:  getEnv("S3_SECRET_KEY", "minio123"),
		S3BucketName: getEnv("S3_BUCKET_NAME", "media-bucket"),
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func (c *Config) String() string {
	return fmt.Sprintf("AppPort=%s, S3Endpoint=%s, S3AccessKey=%s, S3BucketName=%s",
		c.AppPort, c.S3Endpoint, c.S3AccessKey, c.S3BucketName)
}
