package configs

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort    string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPass     string
	DBName     string
	StorageDir string
}

func LoadConfig() *Config {
	return &Config{
		AppPort:    getEnv("MEDIA_APP_PORT", ":8084"),
		DBHost:     getEnv("MEDIA_DB_HOST", "localhost"),
		DBPort:     getEnv("MEDIA_DB_PORT", "5432"),
		DBUser:     getEnv("MEDIA_DB_USER", "postgres"),
		DBPass:     getEnv("MEDIA_DB_PASS", "postgres"),
		DBName:     getEnv("MEDIA_DB_NAME", "media_db"),
		StorageDir: getEnv("MEDIA_STORAGE_DIR", "./uploads"),
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
