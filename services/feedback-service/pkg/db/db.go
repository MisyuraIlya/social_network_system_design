package db

import (
	"log"

	"feedback-gateway/configs"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Db struct {
	DB *gorm.DB
}

func NewDb(cfg *configs.Config) *Db {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	return &Db{DB: db}
}
