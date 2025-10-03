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

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func LoadConfig() *Config {
	return &Config{
		AppPort: env("APP_PORT", ":8081"),
		DBHost:  env("DB_HOST", "user-db"),
		DBPort:  env("DB_PORT", "5432"),
		DBUser:  env("DB_USER", "user"),
		DBPass:  env("DB_PASSWORD", "userpass"),
		DBName:  env("DB_NAME", "user_db"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPass, c.DBName,
	)
}
