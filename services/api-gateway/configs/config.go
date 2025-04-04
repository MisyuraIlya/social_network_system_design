package configs

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort            string
	UserServiceURL     string
	PostServiceURL     string
	MessageServiceURL  string
	MediaServiceURL    string
	FeedServiceURL     string
	FeedbackServiceURL string
}

func LoadConfig() *Config {
	return &Config{
		AppPort:            getEnv("GATEWAY_APP_PORT", ":8080"),
		UserServiceURL:     getEnv("USERS_SERVICE_URL", "http://users-service:8081"),
		PostServiceURL:     getEnv("POST_SERVICE_URL", "http://post-service:8082"),
		MessageServiceURL:  getEnv("MESSAGE_SERVICE_URL", "http://message-service:8086"),
		MediaServiceURL:    getEnv("MEDIA_SERVICE_URL", "http://media-service:8084"),
		FeedServiceURL:     getEnv("FEED_SERVICE_URL", "http://feed-service:8083"),
		FeedbackServiceURL: getEnv("FEEDBACK_SERVICE_URL", "http://feedback-service:8085"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func (c *Config) Print() {
	fmt.Printf("Gateway config: %+v\n", *c)
}
