package configs

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort         string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPass          string
	DBName          string
	RedisHost       string
	RedisPort       string
	KafkaBrokers    string
	KafkaTopic      string
	MediaServiceURL string // URL for the Media Service API
}

func LoadConfig() *Config {
	return &Config{
		AppPort:      getEnv("MESSAGE_APP_PORT", ":8080"),
		DBHost:       getEnv("MESSAGE_DB_HOST", "localhost"),
		DBPort:       getEnv("MESSAGE_DB_PORT", "5432"),
		DBUser:       getEnv("MESSAGE_DB_USER", "postgres"),
		DBPass:       getEnv("MESSAGE_DB_PASS", "postgres"),
		DBName:       getEnv("MESSAGE_DB_NAME", "message_db"),
		RedisHost:    getEnv("MESSAGE_REDIS_HOST", "localhost"),
		RedisPort:    getEnv("MESSAGE_REDIS_PORT", "6379"),
		KafkaBrokers: getEnv("MESSAGE_KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:   getEnv("MESSAGE_KAFKA_TOPIC", "new-message"),
		// Corrected Env Var name and default URL for Media Service API
		MediaServiceURL: getEnv("MEDIA_SERVICE_URL", "http://localhost:8084"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPass, c.DBName,
	)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
