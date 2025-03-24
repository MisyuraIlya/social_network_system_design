package configs

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort string
	DBHost  string
	DBPort  string
	DBUser  string
	DBPass  string
	DBName  string
}

func LoadConfig() *Config {
	return &Config{
		AppPort: getEnv("FEEDBACK_APP_PORT", ":8086"),
		DBHost:  getEnv("FEEDBACK_DB_HOST", "localhost"),
		DBPort:  getEnv("FEEDBACK_DB_PORT", "5432"),
		DBUser:  getEnv("FEEDBACK_DB_USER", "postgres"),
		DBPass:  getEnv("FEEDBACK_DB_PASS", "postgres"),
		DBName:  getEnv("FEEDBACK_DB_NAME", "feedback_db"),
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
