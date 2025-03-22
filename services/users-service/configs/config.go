package configs

import (
	"os"
	"strconv"
)

type Config struct {
	Port int
}

func NewConfig() *Config {
	port := 8080
	if val, ok := os.LookupEnv("PORT"); ok {
		p, err := strconv.Atoi(val)
		if err == nil {
			port = p
		}
	}
	return &Config{Port: port}
}
