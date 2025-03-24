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
		AppPort: getEnv("USER_APP_PORT", ":8081"),
		DBHost:  getEnv("USER_DB_HOST", "localhost"),
		DBPort:  getEnv("USER_DB_PORT", "5432"),
		DBUser:  getEnv("USER_DB_USER", "postgres"),
		DBPass:  getEnv("USER_DB_PASS", "postgres"),
		DBName:  getEnv("USER_DB_NAME", "user_db"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPass, c.DBName,
	)
}
