package configs

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort   string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	RedisHost string
	RedisPort string
}

func LoadConfig() *Config {
	return &Config{
		AppPort:   getEnv("FEEDBACK_APP_PORT", ":8086"),
		DBHost:    getEnv("FEEDBACK_DB_HOST", "localhost"),
		DBPort:    getEnv("FEEDBACK_DB_PORT", "5432"),
		DBUser:    getEnv("FEEDBACK_DB_USER", "postgres"),
		DBPass:    getEnv("FEEDBACK_DB_PASS", "postgres"),
		DBName:    getEnv("FEEDBACK_DB_NAME", "feedback_db"),
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPass, c.DBName,
	)
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
