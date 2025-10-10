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

type DB struct {
	Base   *gorm.DB
	shards map[int]ShardCfg // keep original shard config for logging
}

// ShardInfo returns the shard config for logging.
func (d *DB) ShardInfo(shardID int) (ShardCfg, bool) {
	s, ok := d.shards[shardID]
	return s, ok
}

// RedactDSN returns a safe, short form of a Postgres DSN for logs.
// Example: "host=shard1-pgpool port=5432 dbname=appdb"
func RedactDSN(dsn string) string {
	// Extract host, port, dbname from "key=value" DSN
	reKV := regexp.MustCompile(`\b(host|port|dbname)=\S+`)
	parts := reKV.FindAllString(dsn, -1)
	if len(parts) == 0 {
		// Try URL-form DSN: postgres://user:pass@host:port/dbname?...
		reURL := regexp.MustCompile(`@([^:/\s]+):?(\d+)?/([^?\s]+)`)
		if m := reURL.FindStringSubmatch(dsn); len(m) >= 4 {
			host := m[1]
			port := m[2]
			db := m[3]
			if port == "" {
				return fmt.Sprintf("host=%s dbname=%s", host, db)
			}
			return fmt.Sprintf("host=%s port=%s dbname=%s", host, port, db)
		}
		// Fallback: return first 48 chars
		if len(dsn) > 48 {
			return dsn[:48] + "…"
		}
		return dsn
	}
	return fmt.Sprintf("%s", parts)
}

func OpenFromEnv() *DB {
	raw := os.Getenv("SHARDS_JSON")
	if raw == "" {
		log.Fatal("SHARDS_JSON is required (JSON array of {id, writer, readers[]})")
	}

	var shards []ShardCfg
	if err := json.Unmarshal([]byte(raw), &shards); err != nil || len(shards) == 0 {
		log.Fatalf("invalid SHARDS_JSON: %v", err)
	}

	base, err := openWithRetry(shards[0].Writer, 8, 2*time.Second)
	if err != nil {
		log.Fatalf("open base db failed: %v", err)
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

	first := shards[0]
	firstLabel := fmt.Sprintf("shard%d", first.ID)
	resolver := dbresolver.Register(makeCfg(first), firstLabel)

	for _, s := range shards[1:] {
		label := fmt.Sprintf("shard%d", s.ID)
		resolver = resolver.Register(makeCfg(s), label)
	}

	if err := base.Use(resolver); err != nil {
		log.Fatalf("dbresolver use failed: %v", err)
	}

	shardMap := make(map[int]ShardCfg, len(shards))
	for _, s := range shards {
		shardMap[s.ID] = s
	}

	return &DB{Base: base, shards: shardMap}
}

func (d *DB) Pick(shardID int) *gorm.DB {
	// Reads: hit configured replicas (or pgpool) for this shard.
	return d.Base.Clauses(dbresolver.Use(fmt.Sprintf("shard%d", shardID)))
}

func (d *DB) ForcePrimary(shardID int) *gorm.DB {
	// Writes: force primary (writer) for this shard.
	return d.Base.Clauses(
		dbresolver.Use(fmt.Sprintf("shard%d", shardID)),
		dbresolver.Write,
	)
}

func openWithRetry(dsn string, attempts int, sleep time.Duration) (*gorm.DB, error) {
	var last error
	for i := 1; i <= attempts; i++ {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			sqlDB, err2 := db.DB()
			if err2 == nil && sqlDB != nil {
				pingErr := pingWithTimeout(sqlDB, 2*time.Second)
				if pingErr == nil {
					return db, nil
				}
				last = pingErr
			} else {
				last = err2
			}
		} else {
			last = err
		}

		log.Printf("db open attempt %d/%d failed: %v", i, attempts, last)
		time.Sleep(sleep)
		if sleep < 8*time.Second {
			sleep *= 2
		}
	}
	return nil, last
}

func pingWithTimeout(sqlDB *sql.DB, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- sqlDB.Ping()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("db ping timeout after %s", timeout)
	}
}
