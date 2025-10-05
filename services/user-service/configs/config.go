package configs

import "os"

type Config struct {
	AppPort    string
	NumShards  int    // optional, service also reads directly from env
	ShardsJSON string // optional, service reads this directly too
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func LoadConfig() *Config {
	return &Config{
		AppPort:    env("APP_PORT", ":8081"),
		ShardsJSON: env("SHARDS_JSON", "[]"),
	}
}
