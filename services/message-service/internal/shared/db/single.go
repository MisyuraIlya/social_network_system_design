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
	host := def(os.Getenv("DB_HOST"), "message-db")
	user := def(os.Getenv("DB_USER"), "notify")
	pass := def(os.Getenv("DB_PASSWORD"), "notifypass")
	name := def(os.Getenv("DB_NAME"), "message_db")
	port := def(os.Getenv("DB_PORT"), "5432")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, name)

	base, err := openWithRetry(dsn, 8, 2*time.Second)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	sqlDB, _ := base.DB()
	sqlDB.SetMaxOpenConns(40)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return &Store{Base: base}
}

func def(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func openWithRetry(dsn string, attempts int, sleep time.Duration) (*gorm.DB, error) {
	var last error
	for i := 1; i <= attempts; i++ {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			if s, e := db.DB(); e == nil && s != nil {
				if perr := pingWithTimeout(s, 2*time.Second); perr == nil {
					return db, nil
				} else {
					last = perr
				}
			} else {
				last = e
			}
		} else {
			last = err
		}
		time.Sleep(sleep)
		if sleep < 8*time.Second {
			sleep *= 2
		}
	}
	return nil, last
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
