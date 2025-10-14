package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct{ DB *gorm.DB }

func OpenFromEnv() *Store {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, pass, name, port,
	)

	var last error
	var g *gorm.DB
	for i := 0; i < 8; i++ {
		g, last = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if last == nil {
			sqlDB, _ := g.DB()
			sqlDB.SetMaxOpenConns(40)
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetConnMaxLifetime(30 * time.Minute)
			return &Store{DB: g}
		}
		time.Sleep(time.Duration(1<<i) * time.Second)
	}
	log.Fatalf("db open failed: %v", last)
	return nil
}
