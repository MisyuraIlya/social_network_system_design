package configs

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort string

	RedisHost string
	RedisPort string
	RedisPass string

	KafkaBrokers string
	KafkaTopic   string
	KafkaGroupID string

	UserServiceURL string
}

func LoadConfig() *Config {
	return &Config{
		AppPort:        getEnv("FEED_APP_PORT", ":8085"),
		RedisHost:      getEnv("FEED_REDIS_HOST", "localhost"),
		RedisPort:      getEnv("FEED_REDIS_PORT", "6379"),
		RedisPass:      getEnv("FEED_REDIS_PASS", ""),
		KafkaBrokers:   getEnv("FEED_KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:     getEnv("FEED_KAFKA_TOPIC", "posts"),
		KafkaGroupID:   getEnv("FEED_KAFKA_GROUP_ID", "feed-service-group"),
		UserServiceURL: getEnv("USER_SERVICE_URL", "http://localhost:8081"),
	}
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
