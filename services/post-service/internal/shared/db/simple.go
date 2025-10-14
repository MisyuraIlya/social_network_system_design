package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct{ Base *gorm.DB }

func OpenFromEnv() *Store {
	host := getenv("DB_HOST", "post-db")
	user := getenv("DB_USER", "post")
	pass := getenv("DB_PASSWORD", "postpass")
	name := getenv("DB_NAME", "post_db")
	port := getenv("DB_PORT", "5432")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, name,
	)

	var base *gorm.DB
	var err error
	sleep := time.Second
	for i := 0; i < 8; i++ {
		base, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			sqlDB, _ := base.DB()
			if pingWithTimeout(sqlDB, 2*time.Second) == nil {
				break
			}
		}
		time.Sleep(sleep)
		if sleep < 8*time.Second {
			sleep *= 2
		}
	}
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	sqlDB, _ := base.DB()
	sqlDB.SetMaxOpenConns(40)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return &Store{Base: base}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func pingWithTimeout(sqlDB *sql.DB, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- sqlDB.Ping() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("db ping timeout after %s", timeout)
	}
}
