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
		UserServiceURL:     getEnv("USER_SERVICE_URL", "http://localhost:8081"),
		PostServiceURL:     getEnv("POST_SERVICE_URL", "http://localhost:8082"),
		MessageServiceURL:  getEnv("MESSAGE_SERVICE_URL", "http://localhost:8083"),
		MediaServiceURL:    getEnv("MEDIA_SERVICE_URL", "http://localhost:8084"),
		FeedServiceURL:     getEnv("FEED_SERVICE_URL", "http://localhost:8085"),
		FeedbackServiceURL: getEnv("FEEDBACK_SERVICE_URL", "http://localhost:8086"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// optional helper
func (c *Config) Print() {
	fmt.Printf("Gateway config: %+v\n", *c)
}
