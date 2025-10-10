here my docker compose file
name: socialnet

services:
  #################################################################
  #                           DATABASES                           #
  #################################################################
  # =========================
  # USERS DB – SHARD 1 (repmgr)
  # =========================
  shard1-pg-0:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/postgresql-repmgr:latest
    container_name: shard1-pg-0
    restart: unless-stopped
    environment:
      - POSTGRESQL_USERNAME=app
      - POSTGRESQL_PASSWORD=app_pass_shard1
      - POSTGRESQL_DATABASE=appdb
      - POSTGRESQL_POSTGRES_PASSWORD=supersecret
      - POSTGRESQL_SYNCHRONOUS_COMMIT=on
      - POSTGRESQL_NUM_SYNCHRONOUS_REPLICAS=1
      - REPMGR_PRIMARY_HOST=shard1-pg-0
      - REPMGR_NODE_NAME=shard1-pg-0
      - REPMGR_NODE_NETWORK_NAME=shard1-pg-0
      - REPMGR_PARTNER_NODES=shard1-pg-0,shard1-pg-1,shard1-pg-2
      - REPMGR_USERNAME=repl
      - REPMGR_PASSWORD=repl_pass
      - REPMGR_DATABASE=repmgr
      - REPMGR_PRIMARY_ROLE_WAIT=false
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d appdb"]
      interval: 10s
      timeout: 5s
      retries: 12
    volumes:
      - shard1-pg-0-data:/bitnami/postgresql
    networks: [socialnet]
    ports:
      - "15432:5432"   # admin/debug access to node 0

  shard1-pg-1:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/postgresql-repmgr:latest
    container_name: shard1-pg-1
    restart: unless-stopped
    depends_on: [shard1-pg-0]
    environment:
      - POSTGRESQL_USERNAME=app
      - POSTGRESQL_PASSWORD=app_pass_shard1
      - POSTGRESQL_DATABASE=appdb
      - POSTGRESQL_POSTGRES_PASSWORD=supersecret
      - POSTGRESQL_SYNCHRONOUS_COMMIT=on
      - POSTGRESQL_NUM_SYNCHRONOUS_REPLICAS=1
      - REPMGR_PRIMARY_HOST=shard1-pg-0
      - REPMGR_NODE_NAME=shard1-pg-1
      - REPMGR_NODE_NETWORK_NAME=shard1-pg-1
      - REPMGR_PARTNER_NODES=shard1-pg-0,shard1-pg-1,shard1-pg-2
      - REPMGR_USERNAME=repl
      - REPMGR_PASSWORD=repl_pass
      - REPMGR_DATABASE=repmgr
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d appdb"]
      interval: 10s
      timeout: 5s
      retries: 12
    volumes:
      - shard1-pg-1-data:/bitnami/postgresql
    networks: [socialnet]
    ports:
      - "15433:5432"   # admin/debug access to node 1

  shard1-pg-2:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/postgresql-repmgr:latest
    container_name: shard1-pg-2
    restart: unless-stopped
    depends_on: [shard1-pg-0]
    environment:
      - POSTGRESQL_USERNAME=app
      - POSTGRESQL_PASSWORD=app_pass_shard1
      - POSTGRESQL_DATABASE=appdb
      - POSTGRESQL_POSTGRES_PASSWORD=supersecret
      - POSTGRESQL_SYNCHRONOUS_COMMIT=on
      - POSTGRESQL_NUM_SYNCHRONOUS_REPLICAS=1
      - REPMGR_PRIMARY_HOST=shard1-pg-0
      - REPMGR_NODE_NAME=shard1-pg-2
      - REPMGR_NODE_NETWORK_NAME=shard1-pg-2
      - REPMGR_PARTNER_NODES=shard1-pg-0,shard1-pg-1,shard1-pg-2
      - REPMGR_USERNAME=repl
      - REPMGR_PASSWORD=repl_pass
      - REPMGR_DATABASE=repmgr
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d appdb"]
      interval: 10s
      timeout: 5s
      retries: 12
    volumes:
      - shard1-pg-2-data:/bitnami/postgresql
    networks: [socialnet]
    ports:
      - "15434:5432"   # admin/debug access to node 2

  shard1-pgpool:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/pgpool:latest
    container_name: shard1-pgpool
    restart: unless-stopped
    depends_on:
      - shard1-pg-0
      - shard1-pg-1
      - shard1-pg-2
    environment:
      - PGPOOL_BACKEND_NODES=0:shard1-pg-0:5432,1:shard1-pg-1:5432,2:shard1-pg-2:5432
      - PGPOOL_SR_CHECK_USER=repl
      - PGPOOL_SR_CHECK_PASSWORD=repl_pass
      - PGPOOL_POSTGRES_USERNAME=app
      - PGPOOL_POSTGRES_PASSWORD=app_pass_shard1
      - PGPOOL_ENABLE_LOAD_BALANCING=yes
    healthcheck:
      test: ["CMD-SHELL", "</dev/tcp/127.0.0.1/5432"]
      interval: 15s
      timeout: 3s
      retries: 10
    networks: [socialnet]
    ports:
      - "6433:5432"    # EXPOSED: client entrypoint for Shard 1

  # =========================
  # USERS DB – SHARD 2 (repmgr)
  # =========================
  shard2-pg-0:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/postgresql-repmgr:latest
    container_name: shard2-pg-0
    restart: unless-stopped
    environment:
      - POSTGRESQL_USERNAME=app
      - POSTGRESQL_PASSWORD=app_pass_shard2
      - POSTGRESQL_DATABASE=appdb
      - POSTGRESQL_POSTGRES_PASSWORD=supersecret
      - POSTGRESQL_SYNCHRONOUS_COMMIT=on
      - POSTGRESQL_NUM_SYNCHRONOUS_REPLICAS=1
      - REPMGR_PRIMARY_HOST=shard2-pg-0
      - REPMGR_NODE_NAME=shard2-pg-0
      - REPMGR_NODE_NETWORK_NAME=shard2-pg-0
      - REPMGR_PARTNER_NODES=shard2-pg-0,shard2-pg-1,shard2-pg-2
      - REPMGR_USERNAME=repl
      - REPMGR_PASSWORD=repl_pass
      - REPMGR_DATABASE=repmgr
      - REPMGR_PRIMARY_ROLE_WAIT=false
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d appdb"]
      interval: 10s
      timeout: 5s
      retries: 12
    volumes:
      - shard2-pg-0-data:/bitnami/postgresql
    networks: [socialnet]
    ports:
      - "25432:5432"   # admin/debug access to node 0

  shard2-pg-1:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/postgresql-repmgr:latest
    container_name: shard2-pg-1
    restart: unless-stopped
    depends_on: [shard2-pg-0]
    environment:
      - POSTGRESQL_USERNAME=app
      - POSTGRESQL_PASSWORD=app_pass_shard2
      - POSTGRESQL_DATABASE=appdb
      - POSTGRESQL_POSTGRES_PASSWORD=supersecret
      - POSTGRESQL_SYNCHRONOUS_COMMIT=on
      - POSTGRESQL_NUM_SYNCHRONOUS_REPLICAS=1
      - REPMGR_PRIMARY_HOST=shard2-pg-0
      - REPMGR_NODE_NAME=shard2-pg-1
      - REPMGR_NODE_NETWORK_NAME=shard2-pg-1
      - REPMGR_PARTNER_NODES=shard2-pg-0,shard2-pg-1,shard2-pg-2
      - REPMGR_USERNAME=repl
      - REPMGR_PASSWORD=repl_pass
      - REPMGR_DATABASE=repmgr
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d appdb"]
      interval: 10s
      timeout: 5s
      retries: 12
    volumes:
      - shard2-pg-1-data:/bitnami/postgresql
    networks: [socialnet]
    ports:
      - "25433:5432"   # admin/debug access to node 1

  shard2-pg-2:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/postgresql-repmgr:latest
    container_name: shard2-pg-2
    restart: unless-stopped
    depends_on: [shard2-pg-0]
    environment:
      - POSTGRESQL_USERNAME=app
      - POSTGRESQL_PASSWORD=app_pass_shard2
      - POSTGRESQL_DATABASE=appdb
      - POSTGRESQL_POSTGRES_PASSWORD=supersecret
      - POSTGRESQL_SYNCHRONOUS_COMMIT=on
      - POSTGRESQL_NUM_SYNCHRONOUS_REPLICAS=1
      - REPMGR_PRIMARY_HOST=shard2-pg-0
      - REPMGR_NODE_NAME=shard2-pg-2
      - REPMGR_NODE_NETWORK_NAME=shard2-pg-2
      - REPMGR_PARTNER_NODES=shard2-pg-0,shard2-pg-1,shard2-pg-2
      - REPMGR_USERNAME=repl
      - REPMGR_PASSWORD=repl_pass
      - REPMGR_DATABASE=repmgr
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d appdb"]
      interval: 10s
      timeout: 5s
      retries: 12
    volumes:
      - shard2-pg-2-data:/bitnami/postgresql
    networks: [socialnet]
    ports:
      - "25434:5432"   # admin/debug access to node 2

  shard2-pgpool:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/pgpool:latest
    container_name: shard2-pgpool
    restart: unless-stopped
    depends_on:
      - shard2-pg-0
      - shard2-pg-1
      - shard2-pg-2
    environment:
      - PGPOOL_BACKEND_NODES=0:shard2-pg-0:5432,1:shard2-pg-1:5432,2:shard2-pg-2:5432
      - PGPOOL_SR_CHECK_USER=repl
      - PGPOOL_SR_CHECK_PASSWORD=repl_pass
      - PGPOOL_POSTGRES_USERNAME=app
      - PGPOOL_POSTGRES_PASSWORD=app_pass_shard2
      - PGPOOL_ENABLE_LOAD_BALANCING=yes
    healthcheck:
      test: ["CMD-SHELL", "</dev/tcp/127.0.0.1/5432"]
      interval: 15s
      timeout: 3s
      retries: 10
    networks: [socialnet]
    ports:
      - "7433:5432"    # EXPOSED: client entrypoint for Shard 2

  # -------------------------
  # OTHER Postgres DBs
  # -------------------------
  post-db:
    image: postgres:15
    container_name: post-db
    environment:
      POSTGRES_USER: post
      POSTGRES_PASSWORD: postpass
      POSTGRES_DB: post_db
    volumes:
      - post_db_data:/var/lib/postgresql/data
    networks: [socialnet]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U post -d post_db"]
      interval: 5s
      timeout: 3s
      retries: 30
    ports:
      - "5434:5432"

  feedback-db:
    image: postgres:15
    container_name: feedback-db
    environment:
      POSTGRES_USER: feedback
      POSTGRES_PASSWORD: feedbackpass
      POSTGRES_DB: feedback_db
    volumes:
      - feedback_db_data:/var/lib/postgresql/data
    networks: [socialnet]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U feedback -d feedback_db"]
      interval: 5s
      timeout: 3s
      retries: 30
    ports:
      - "5435:5432"

  message-db:
    image: postgres:15
    container_name: message-db
    environment:
      POSTGRES_USER: notify
      POSTGRES_PASSWORD: notifypass
      POSTGRES_DB: message_db
    volumes:
      - message_db_data:/var/lib/postgresql/data
    networks: [socialnet]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U notify -d message_db"]
      interval: 5s
      timeout: 3s
      retries: 30
    ports:
      - "5436:5432"

  #################################################################
  #                           CACHES / QUEUES                     #
  #################################################################
  redis-feed:
    image: redis:7
    container_name: redis-feed
    command: ["redis-server", "--appendonly", "yes"]
    volumes:
      - redis_feed_data:/data
    networks: [socialnet]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 30

  redis-message:
    image: redis:7
    container_name: redis-message
    command: ["redis-server", "--appendonly", "yes"]
    volumes:
      - redis_message_data:/data
    networks: [socialnet]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 30

  redis-feedback:
    image: redis:7
    container_name: redis-feedback
    command: ["redis-server", "--appendonly", "yes"]
    volumes:
      - redis_feedback_data:/data
    networks: [socialnet]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 30

  kafka:
    image: confluentinc/cp-kafka:7.7.1
    container_name: kafka
    ports:
      - "9092:9092"
    environment:
      KAFKA_NODE_ID: "1"
      KAFKA_PROCESS_ROLES: "broker,controller"
      KAFKA_LISTENERS: "PLAINTEXT://:9092,CONTROLLER://:9093"
      KAFKA_ADVERTISED_LISTENERS: "PLAINTEXT://kafka:9092"
      KAFKA_CONTROLLER_LISTENER_NAMES: "CONTROLLER"
      KAFKA_CONTROLLER_QUORUM_VOTERS: "1@kafka:9093"
      KAFKA_INTER_BROKER_LISTENER_NAME: "PLAINTEXT"
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: "1"
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: "1"
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: "1"
      CLUSTER_ID: "MkU3OEVBNTcwNTJENDM2Qk"
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: "0"
    networks: [socialnet]
    volumes:
      - kafka_data:/var/lib/kafka/data
    healthcheck:
      test: ["CMD-SHELL", "kafka-topics --bootstrap-server localhost:9092 --list >/dev/null 2>&1 || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 30
      start_period: 20s

  #################################################################
  #                           MINIO / S3                          #
  #################################################################
  minio:
    image: minio/minio:latest
    container_name: minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minio
      MINIO_ROOT_PASSWORD: minio123
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    networks: [socialnet]
    healthcheck:
      test: ["CMD-SHELL", "curl -s http://localhost:9000/minio/health/ready || exit 1"]
      interval: 5s
      timeout: 3s
      retries: 30

  #################################################################
  #                       USER SERVICE (N SHARDS)                 #
  #################################################################
  user-service:
    build:
      context: ./services/user-service
    container_name: user-service
    volumes:
      - ./services/user-service:/app
    environment:
      APP_PORT: ":8081"
      NUM_SHARDS: "2"
      SHARDS_JSON: >
        [
          {"id":0,"dsn":"host=shard1-pgpool port=5432 user=app password=app_pass_shard1 dbname=appdb sslmode=disable"},
          {"id":1,"dsn":"host=shard2-pgpool port=5432 user=app password=app_pass_shard2 dbname=appdb sslmode=disable"}
        ]
      AIR_WATCHER_FORCE_POLLING: "true"
      AIR_TMP_DIR: "/app/tmp"
    depends_on:
      shard1-pgpool:
        condition: service_started
      shard2-pgpool:
        condition: service_started
    networks: [socialnet]
    ports:
      - "8081:8081"

  #################################################################
  #                   OTHER CORE MICROSERVICES                    #
  #################################################################
  post-service:
    container_name: post-service
    build: ./services/post-service
    environment:
      DB_HOST: post-db
      DB_USER: post
      DB_PASSWORD: postpass
      DB_NAME: post_db
      MEDIA_SERVICE_URL: http://minio:9000
      KAFKA_BOOTSTRAP_SERVERS: kafka:9092
    depends_on:
      post-db:
        condition: service_healthy
      kafka:
        condition: service_healthy
      minio:
        condition: service_healthy
    networks: [socialnet]
    ports:
      - "8082:8082"

  feed-service:
    container_name: feed-service
    build:
      context: ./services/feed-service
      dockerfile: Dockerfile
    environment:
      REDIS_HOST: redis-feed
      REDIS_PORT: 6379
      KAFKA_BOOTSTRAP_SERVERS: kafka:9092
    depends_on:
      redis-feed:
        condition: service_healthy
      kafka:
        condition: service_healthy
    networks: [socialnet]
    ports:
      - "8083:8083"

  feedback-service:
    container_name: feedback-service
    build: ./services/feedback-service
    environment:
      DB_HOST: feedback-db
      DB_USER: feedback
      DB_PASSWORD: feedbackpass
      DB_NAME: feedback_db
      REDIS_HOST: redis-feedback
      REDIS_PORT: 6379
    depends_on:
      feedback-db:
        condition: service_healthy
      redis-feedback:
        condition: service_healthy
    networks: [socialnet]
    ports:
      - "8084:8084"

  message-service:
    container_name: message-service
    build: ./services/message-service
    environment:
      DB_HOST: message-db
      DB_USER: notify
      DB_PASSWORD: notifypass
      DB_NAME: message_db
      KAFKA_BOOTSTRAP_SERVERS: kafka:9092
      REDIS_HOST: redis-message
      REDIS_PORT: 6379
    depends_on:
      message-db:
        condition: service_healthy
      kafka:
        condition: service_healthy
      redis-message:
        condition: service_healthy
    networks: [socialnet]
    ports:
      - "8085:8085"

  notification-service:
    container_name: notification-service
    build: ./services/notification-service
    environment:
      KAFKA_BOOTSTRAP_SERVERS: kafka:9092
    depends_on:
      kafka:
        condition: service_healthy
    networks: [socialnet]
    ports:
      - "8086:8086"

  media-service:
    container_name: media-service
    build: ./services/media-service
    environment:
      S3_ENDPOINT: http://minio:9000
      S3_ACCESS_KEY: minio
      S3_SECRET_KEY: minio123
    depends_on:
      minio:
        condition: service_healthy
    networks: [socialnet]
    ports:
      - "8088:8088"

  #################################################################
  #                   API GATEWAY / LOAD BALANCER                 #
  #################################################################
  loadbalancer:
    image: nginx:latest
    container_name: loadbalancer
    depends_on:
      - api-gateway
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    ports:
      - "80:80"
    networks: [socialnet]

  api-gateway:
    container_name: api-gateway
    build: ./services/api-gateway
    environment:
      USER_SERVICE_URL: http://user-service:8081
      POST_SERVICE_URL: http://post-service:8082
      FEED_SERVICE_URL: http://feed-service:8083
      FEEDBACK_SERVICE_URL: http://feedback-service:8084
      MESSAGE_SERVICE_URL: http://message-service:8085
      MEDIA_SERVICE_URL: http://media-service:8088
      NOTIFICATION_SERVICE_URL: http://notification-service:8086
    depends_on:
      user-service:
        condition: service_started
      post-service:
        condition: service_started
      feed-service:
        condition: service_started
      feedback-service:
        condition: service_started
      message-service:
        condition: service_started
      media-service:
        condition: service_started
      notification-service:
        condition: service_started
    networks: [socialnet]
    ports:
      - "8080:8080"

  #################################################################
  #                    SWAGGER UI FOR DOCS                        #
  #################################################################
  swagger-ui:
    image: swaggerapi/swagger-ui:latest
    container_name: swagger-ui
    depends_on:
      - api-gateway
    ports:
      - "9002:8080"
    environment:
      SWAGGER_JSON: /openapi/combined_openapi.yaml
    volumes:
      - ./combined_openapi.yaml:/openapi/combined_openapi.yaml:ro
    networks: [socialnet]

networks:
  socialnet:

volumes:
  # Users DB shards (HA)
  shard1-pg-0-data:
  shard1-pg-1-data:
  shard1-pg-2-data:
  shard2-pg-0-data:
  shard2-pg-1-data:
  shard2-pg-2-data:

  # Other Postgres
  post_db_data:
  feedback_db_data:
  message_db_data:

  # Redis
  redis_feed_data:
  redis_message_data:
  redis_feedback_data:

  # MinIO
  minio_data:

  # Kafka
  kafka_data:


i currently work on users service
here my code
services/user-service/cmd/main.go
package main

import (
	"fmt"
	"log"
	"net/http"

	"users-service/internal/user"
	multidb "users-service/pkg/db"

	"gorm.io/gorm"
)

func main() {
	// Open all shard connections from SHARDS_JSON
	mdb := multidb.OpenMultiFromEnv()

	// Auto-migrate the users table on every shard
	if err := mdb.Range(func(id int, db *gorm.DB) error {
		return db.AutoMigrate(&user.User{})
	}); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Wire repository/service/handler
	repo := user.NewUserRepository(mdb) // multi-shard aware
	svc := user.NewUserService(repo)    // picks shard per request
	handler := user.NewUserHandler(svc) // HTTP

	mux := http.NewServeMux()
	user.RegisterRoutes(mux, handler)

	addr := ":8081"
	fmt.Printf("User Service listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
services/user-service/configs/config.go
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

services/user-service/pkg/db/multi.go
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

services/user-service/pkg/shard/shard.go
package shard

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"
)

func PickShard(key string, numShards int) int {
	h := sha256.Sum256([]byte(key))
	v := binary.BigEndian.Uint32(h[:4]) ^ binary.BigEndian.Uint32(h[4:8])
	return int(uint32(v) % uint32(numShards))
}

// "0-abcdef..." → 0, true
func ExtractShard(userID string) (int, bool) {
	i := strings.IndexByte(userID, '-')
	if i <= 0 {
		return 0, false
	}
	n, err := strconv.Atoi(userID[:i])
	if err != nil {
		return 0, false
	}
	return n, true
}

services/user-service/pkg/req/decode.go
package req

import (
	"encoding/json"
	"io"
)

func Decode[T any](body io.ReadCloser) (T, error) {
	var payload T
	err := json.NewDecoder(body).Decode(&payload)
	if err != nil {
		return payload, err
	}
	return payload, nil
}

services/user-service/pkg/req/handle.go
package req

import (
	"net/http"
	"users-service/pkg/res"
)

func HandleBody[T any](w *http.ResponseWriter, r *http.Request) (*T, error) {
	body, err := Decode[T](r.Body)
	if err != nil {
		res.Json(*w, err.Error(), 402)
		return nil, err
	}
	err = IsValid(body)
	if err != nil {
		res.Json(*w, err.Error(), 402)
		return nil, err
	}
	return &body, nil
}

services/user-service/pkg/req/validate.go
package req

import (
	"github.com/go-playground/validator/v10"
)

func IsValid[T any](payload T) error {
	validate := validator.New()
	err := validate.Struct(payload)
	return err
}

services/user-service/pkg/res/res.go
package res

import (
	"encoding/json"
	"net/http"
)

func Json(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

services/user-service/internal/user/handler.go
package user

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"users-service/internal/auth"
)

type UserHandler struct {
	Service IUserService
}

func NewUserHandler(svc IUserService) *UserHandler { return &UserHandler{Service: svc} }

func RegisterRoutes(mux *http.ServeMux, h *UserHandler) {
	// POST /users  -> register
	// GET  /users  -> list MY shard (requires JWT)
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.Register(w, r)
		case http.MethodGet:
			h.ListMine(w, r) // <-- lists only the caller's shard via JWT
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	// POST /users/login -> login + returns JWT
	mux.HandleFunc("/users/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		h.Login(w, r)
	})

	// GET /users/{user_id} -> fetch by id (service derives shard from user_id prefix)
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 2 {
			http.Error(w, "User ID missing", http.StatusBadRequest)
			return
		}
		h.GetUser(w, r, parts[1])
	})

	// OPTIONAL admin/dev route to peek a specific shard
	mux.HandleFunc("/admin/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		h.ListAdminByShard(w, r)
	})
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	u, err := h.Service.Register(body.Email, body.Password, body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// issue a JWT on register as well (handy for immediate login)
	tok, _ := auth.MakeJWT(u.UserID, u.ShardID)

	w.Header().Set("X-Shard-ID", strconv.Itoa(u.ShardID)) // debug only
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"user_id":      u.UserID,
		"email":        u.Email,
		"name":         u.Name,
		"access_token": tok,
	})
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	u, err := h.Service.Login(body.Email, body.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tok, _ := auth.MakeJWT(u.UserID, u.ShardID)

	w.Header().Set("X-Shard-ID", strconv.Itoa(u.ShardID)) // debug only
	json.NewEncoder(w).Encode(map[string]any{
		"message":      "login successful",
		"user_id":      u.UserID,
		"name":         u.Name,
		"email":        u.Email,
		"access_token": tok,
	})
}

func (h *UserHandler) GetUser(w http.ResponseWriter, _ *http.Request, userID string) {
	usr, err := h.Service.GetByUserID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(usr)
}

// GET /users -> lists MY shard, requires Authorization: Bearer <JWT>
// Supports optional ?limit=&offset=
func (h *UserHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	_, shardID, err := auth.ParseAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	users, err := h.Service.ListShard(shardID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"shard_id": shardID,
		"limit":    limit,
		"offset":   offset,
		"items":    users,
	})
}

// OPTIONAL admin/dev: GET /admin/users?shard=0&limit=50&offset=0
func (h *UserHandler) ListAdminByShard(w http.ResponseWriter, r *http.Request) {
	shStr := r.URL.Query().Get("shard")
	if shStr == "" {
		http.Error(w, "shard param required", http.StatusBadRequest)
		return
	}
	shardID, err := strconv.Atoi(shStr)
	if err != nil {
		http.Error(w, "invalid shard", http.StatusBadRequest)
		return
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	users, err := h.Service.ListShard(shardID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"shard_id": shardID,
		"limit":    limit,
		"offset":   offset,
		"items":    users,
	})
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

services/user-service/internal/user/model.go
package user

import "time"

type User struct {
	UserID   string `gorm:"uniqueIndex;size:64" json:"user_id"`
	ShardID  int    `gorm:"index" json:"shard_id"`
	ID       uint   `gorm:"primaryKey" json:"-"`
	Email    string `gorm:"uniqueIndex;size:100" json:"email"`
	Password string `gorm:"size:255" json:"-"`
	Name     string `gorm:"size:100" json:"name"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

services/user-service/internal/user/payload.go
package user

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required"`
}

type RegisterResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

services/user-service/internal/user/repository.go
package user

import (
	"errors"

	"users-service/pkg/shard"

	"gorm.io/gorm"
)

type ShardedDB interface {
	Get(shardID int) *gorm.DB
}

type IUserRepository interface {
	Create(u *User) (*User, error)
	FindByEmail(email string, shardID int) (*User, error)
	FindByUserID(uid string) (*User, error)
	FindAllByShard(shardID int) ([]User, error)
	FindAllByShardPaged(shardID, limit, offset int) ([]User, error)
}

type UserRepository struct {
	mdb ShardedDB
}

func NewUserRepository(mdb ShardedDB) IUserRepository {
	return &UserRepository{mdb: mdb}
}

func (r *UserRepository) Create(u *User) (*User, error) {
	if err := r.mdb.Get(u.ShardID).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(email string, shardID int) (*User, error) {
	var u User
	if err := r.mdb.Get(shardID).Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByUserID(uid string) (*User, error) {
	sh, ok := shard.ExtractShard(uid)
	if !ok {
		return nil, errors.New("invalid user_id format")
	}
	var u User
	if err := r.mdb.Get(sh).Where("user_id = ?", uid).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindAllByShard(shardID int) ([]User, error) {
	var users []User
	if err := r.mdb.Get(shardID).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) FindAllByShardPaged(shardID, limit, offset int) ([]User, error) {
	var users []User
	if err := r.mdb.Get(shardID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

services/user-service/internal/user/service.go
package user

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"

	"users-service/pkg/shard"
)

type IUserService interface {
	Register(email, password, name string) (*User, error)
	Login(email, password string) (*User, error)
	ListAll(shardID int) ([]User, error)
	ListShard(shardID, limit, offset int) ([]User, error)
	GetByUserID(uid string) (*User, error)
}

type UserService struct {
	repo      IUserRepository
	numShards int
}

func NewUserService(repo IUserRepository) IUserService {
	ns := 1
	if v := os.Getenv("NUM_SHARDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ns = n
		}
	}
	return &UserService{repo: repo, numShards: ns}
}

func (s *UserService) Register(email, password, name string) (*User, error) {
	sh := shard.PickShard(email, s.numShards)

	// ensure uniqueness on the owning shard
	if existing, _ := s.repo.FindByEmail(email, sh); existing != nil {
		return nil, errors.New("user already exists")
	}

	// user_id format: "<shard>-<random64hex>"
	var b [8]byte
	_, _ = rand.Read(b[:])
	uid := fmt.Sprintf("%d-%x", sh, binary.BigEndian.Uint64(b[:]))

	u := &User{
		UserID:   uid,
		ShardID:  sh,
		Email:    email,
		Password: password, // TODO: bcrypt
		Name:     name,
	}
	return s.repo.Create(u)
}

func (s *UserService) Login(email, password string) (*User, error) {
	sh := shard.PickShard(email, s.numShards)
	usr, err := s.repo.FindByEmail(email, sh)
	if err != nil || usr.Password != password {
		return nil, errors.New("wrong credentials")
	}
	return usr, nil
}

func (s *UserService) ListAll(shardID int) ([]User, error) {
	return s.repo.FindAllByShard(shardID)
}

func (s *UserService) GetByUserID(uid string) (*User, error) {
	return s.repo.FindByUserID(uid)
}

func (s *UserService) ListShard(shardID, limit, offset int) ([]User, error) {
	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.FindAllByShardPaged(shardID, limit, offset)
}

services/user-service/internal/auth/jwt.go
package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("replace-this-with-a-strong-secret")

func MakeJWT(userID string, shardID int) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"sh":  shardID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(jwtKey)
}

func ParseAuthHeader(authz string) (userID string, shardID int, err error) {
	if authz == "" {
		return "", 0, errors.New("missing Authorization")
	}
	tokenStr := strings.TrimPrefix(authz, "Bearer ")
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return jwtKey, nil
	})
	if err != nil || !tok.Valid {
		return "", 0, errors.New("invalid token")
	}
	mc, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", 0, errors.New("bad claims")
	}
	uid, _ := mc["sub"].(string)
	// sh comes as float64 from JSON numbers
	shf, ok := mc["sh"].(float64)
	if !ok {
		return "", 0, errors.New("missing shard claim")
	}
	return uid, int(shf), nil
}