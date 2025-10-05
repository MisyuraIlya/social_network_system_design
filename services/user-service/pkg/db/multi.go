package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ShardCfg struct {
	ID  int    `json:"id"`
	DSN string `json:"dsn"`
}

type Multi struct {
	dbs map[int]*gorm.DB
	mu  sync.RWMutex
}

func OpenMultiFromEnv() *Multi {
	raw := os.Getenv("SHARDS_JSON")
	if raw == "" {
		log.Fatal("SHARDS_JSON is required (JSON array of {id, dsn})")
	}
	var shards []ShardCfg
	if err := json.Unmarshal([]byte(raw), &shards); err != nil {
		log.Fatalf("invalid SHARDS_JSON: %v", err)
	}
	if len(shards) == 0 {
		log.Fatal("SHARDS_JSON is empty")
	}

	m := &Multi{dbs: make(map[int]*gorm.DB)}
	for _, s := range shards {
		db, err := gorm.Open(postgres.Open(s.DSN), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect shard %d: %v", s.ID, err)
		}
		m.dbs[s.ID] = db
	}
	return m
}

func (m *Multi) Get(shardID int) *gorm.DB {
	m.mu.RLock()
	db := m.dbs[shardID]
	m.mu.RUnlock()
	if db == nil {
		panic(fmt.Sprintf("no db for shard %d", shardID))
	}
	return db
}

func (m *Multi) Range(fn func(id int, db *gorm.DB) error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for id, db := range m.dbs {
		if err := fn(id, db); err != nil {
			return err
		}
	}
	return nil
}
