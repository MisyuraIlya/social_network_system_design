package db

import (
	"log"

	"post-service/configs"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Db struct {
	DB *gorm.DB
}

func NewDb(cfg *configs.Config) *Db {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Could not connect to Postgres: %v", err)
	}
	return &Db{DB: db}
}
