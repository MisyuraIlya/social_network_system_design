package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

type ShardCfg struct {
	ID      int      `json:"id"`
	Writer  string   `json:"writer"`
	Readers []string `json:"readers,omitempty"`
}

type Store struct {
	Base   *gorm.DB
	shards map[int]ShardCfg
}

func (s *Store) Use(shardID int) *gorm.DB {
	return s.Base.Clauses(dbresolver.Use(fmt.Sprintf("shard%d", shardID)))
}
func (s *Store) Write(shardID int) *gorm.DB {
	return s.Base.Clauses(
		dbresolver.Use(fmt.Sprintf("shard%d", shardID)),
		dbresolver.Write,
	)
}
func (s *Store) ShardInfo(id int) (ShardCfg, bool) { c, ok := s.shards[id]; return c, ok }

func OpenFromEnv() *Store {
	raw := os.Getenv("SHARDS_JSON")
	if raw == "" {
		log.Fatal("SHARDS_JSON not set")
	}

	var shards []ShardCfg
	if err := json.Unmarshal([]byte(raw), &shards); err != nil || len(shards) == 0 {
		log.Fatalf("invalid SHARDS_JSON: %v", err)
	}

	base, err := openWithRetry(shards[0].Writer, 8, 2*time.Second)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}

	sqlDB, _ := base.DB()
	sqlDB.SetMaxOpenConns(40)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	makeCfg := func(s ShardCfg) dbresolver.Config {
		var readers []gorm.Dialector
		for _, r := range s.Readers {
			readers = append(readers, postgres.Open(r))
		}
		return dbresolver.Config{
			Sources:  []gorm.Dialector{postgres.Open(s.Writer)},
			Replicas: readers,
			Policy:   dbresolver.RandomPolicy{},
		}
	}

	r := dbresolver.Register(makeCfg(shards[0]), fmt.Sprintf("shard%d", shards[0].ID))
	for _, s := range shards[1:] {
		r = r.Register(makeCfg(s), fmt.Sprintf("shard%d", s.ID))
	}
	if err := base.Use(r); err != nil {
		log.Fatalf("dbresolver: %v", err)
	}

	imap := make(map[int]ShardCfg, len(shards))
	for _, s := range shards {
		imap[s.ID] = s
	}
	return &Store{Base: base, shards: imap}
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

// Redact a few DSN kvs in logs (best-effort)
var reKV = regexp.MustCompile(`\b(host|port|dbname)=\S+`)

func RedactDSN(dsn string) string {
	parts := reKV.FindAllString(dsn, -1)
	if len(parts) == 0 {
		if len(dsn) > 48 {
			return dsn[:48] + "…"
		}
		return dsn
	}
	return fmt.Sprintf("%s", parts)
}
