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
    networks:
      socialnet: {}
    ports:
      - "15432:5432"

  shard1-pg-1:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/postgresql-repmgr:latest
    container_name: shard1-pg-1
    restart: unless-stopped
    depends_on:
      shard1-pg-0:
        condition: service_started
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
    networks:
      socialnet: {}
    ports:
      - "15433:5432"

  shard1-pg-2:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/postgresql-repmgr:latest
    container_name: shard1-pg-2
    restart: unless-stopped
    depends_on:
      shard1-pg-0:
        condition: service_started
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
    networks:
      socialnet: {}
    ports:
      - "15434:5432"

  shard1-pgpool:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/pgpool:latest
    container_name: shard1-pgpool
    restart: unless-stopped
    depends_on:
      shard1-pg-0:
        condition: service_healthy
      shard1-pg-1:
        condition: service_healthy
      shard1-pg-2:
        condition: service_healthy
    environment:
      - PGPOOL_BACKEND_NODES=0:shard1-pg-0:5432,1:shard1-pg-1:5432,2:shard1-pg-2:5432
      - PGPOOL_SR_CHECK_USER=repl
      - PGPOOL_SR_CHECK_PASSWORD=repl_pass
      - PGPOOL_POSTGRES_USERNAME=app
      - PGPOOL_POSTGRES_PASSWORD=app_pass_shard1
      - PGPOOL_ENABLE_LOAD_BALANCING=yes
      - PGPOOL_ADMIN_USERNAME=admin
      - PGPOOL_ADMIN_PASSWORD=adminpass
    healthcheck:
      test: ["CMD-SHELL", "bash -lc '</dev/tcp/127.0.0.1/5432' || exit 1"]
    networks:
      socialnet:
        aliases: [shard1-pgpool]
    ports:
      - "6433:5432"

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
    volumes:
      - shard2-pg-0-data:/bitnami/postgresql
    networks:
      socialnet: {}
    ports:
      - "25432:5432"

  shard2-pg-1:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/postgresql-repmgr:latest
    container_name: shard2-pg-1
    restart: unless-stopped
    depends_on:
      shard2-pg-0:
        condition: service_started
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
    volumes:
      - shard2-pg-1-data:/bitnami/postgresql
    networks:
      socialnet: {}
    ports:
      - "25433:5432"

  shard2-pg-2:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/postgresql-repmgr:latest
    container_name: shard2-pg-2
    restart: unless-stopped
    depends_on:
      shard2-pg-0:
        condition: service_started
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
    volumes:
      - shard2-pg-2-data:/bitnami/postgresql
    networks:
      socialnet: {}
    ports:
      - "25434:5432"

  shard2-pgpool:
    platform: linux/amd64
    image: public.ecr.aws/bitnami/pgpool:latest
    container_name: shard2-pgpool
    restart: unless-stopped
    depends_on:
      shard2-pg-0:
        condition: service_healthy
      shard2-pg-1:
        condition: service_healthy
      shard2-pg-2:
        condition: service_healthy
    environment:
      - PGPOOL_BACKEND_NODES=0:shard2-pg-0:5432,1:shard2-pg-1:5432,2:shard2-pg-2:5432
      - PGPOOL_SR_CHECK_USER=repl
      - PGPOOL_SR_CHECK_PASSWORD=repl_pass
      - PGPOOL_POSTGRES_USERNAME=app
      - PGPOOL_POSTGRES_PASSWORD=app_pass_shard2
      - PGPOOL_ENABLE_LOAD_BALANCING=yes
      - PGPOOL_ADMIN_USERNAME=admin
      - PGPOOL_ADMIN_PASSWORD=adminpass
    healthcheck:
      test: ["CMD-SHELL", "bash -lc '</dev/tcp/127.0.0.1/5432' || exit 1"]
    networks:
      socialnet:
        aliases: [shard2-pgpool]
    ports:
      - "7433:5432"

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
    networks:
      socialnet: {}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U post -d post_db"]
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
    networks:
      socialnet: {}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U feedback -d feedback_db"]
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
    networks:
      socialnet: {}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U notify -d message_db"]
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
    networks:
      socialnet: {}
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]

  redis-message:
    image: redis:7
    container_name: redis-message
    command: ["redis-server", "--appendonly", "yes"]
    volumes:
      - redis_message_data:/data
    networks:
      socialnet: {}
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]

  redis-feedback:
    image: redis:7
    container_name: redis-feedback
    command: ["redis-server", "--appendonly", "yes"]
    volumes:
      - redis_feedback_data:/data
    networks:
      socialnet: {}
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]

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
    networks:
      socialnet: {}
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
    networks:
      socialnet: {}
    healthcheck:
      test: ["CMD-SHELL", "curl -s http://localhost:9000/minio/health/ready || exit 1"]

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
          {"id":0,
          "writer":"host=shard1-pgpool port=5432 user=app password=app_pass_shard1 dbname=appdb sslmode=disable",
          "readers":[
            "host=shard1-pgpool port=5432 user=app password=app_pass_shard1 dbname=appdb sslmode=disable"
          ]},
          {"id":1,
          "writer":"host=shard2-pgpool port=5432 user=app password=app_pass_shard2 dbname=appdb sslmode=disable",
          "readers":[
            "host=shard2-pgpool port=5432 user=app password=app_pass_shard2 dbname=appdb sslmode=disable"
          ]}
        ]
      JWT_SECRET: "super-long-random-secret"
      AUTO_MIGRATE: "true"
      AIR_WATCHER_FORCE_POLLING: "true"
      AIR_TMP_DIR: "/app/tmp"
      # OpenTelemetry
      OTEL_EXPORTER_OTLP_ENDPOINT: "otel-collector:4318"
      OTEL_TRACES_SAMPLER: "parentbased_traceidratio"
      OTEL_TRACES_SAMPLER_ARG: "1.0"
      OTEL_RESOURCE_ATTRIBUTES: "service.name=user-service,service.version=1.0.0,env=local"
      OTEL_SERVICE_NAME: "user-service"
      OTEL_EXPORTER_OTLP_TRACES_ENDPOINT: "http://otel-collector:4318/v1/traces"
      OTEL_EXPORTER_OTLP_PROTOCOL: "http/protobuf"
      OTEL_EXPORTER_OTLP_TRACES_PROTOCOL: "http/protobuf"
    depends_on:
      shard1-pgpool:
        condition: service_healthy
      shard2-pgpool:
        condition: service_healthy
      otel-collector:
        condition: service_started
    networks:
      socialnet: {}
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
    networks:
      socialnet: {}
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
    networks:
      socialnet: {}
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
    networks:
      socialnet: {}
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
    networks:
      socialnet: {}
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
    networks:
      socialnet: {}
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
    networks:
      socialnet: {}
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
    networks:
      socialnet: {}

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
    networks:
      socialnet: {}
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
    networks:
      socialnet: {}

  #################################################################
  #                        OBSERVABILITY                          #
  #################################################################
  otel-collector:
    image: otel/opentelemetry-collector:0.106.0
    container_name: otel-collector
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./observability/otel-collector-config.yaml:/etc/otel-collector-config.yaml:ro
    ports:
      - "4317:4317"
      - "4318:4318"
    networks:
      socialnet: {}

  jaeger:
    image: jaegertracing/all-in-one:1.59
    container_name: jaeger
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    ports:
      - "16686:16686"
      - "14250:14250"
      - "14268:14268"
    networks:
      socialnet: {}

  loki:
    image: grafana/loki:2.9.8
    container_name: loki
    command: ["-config.file=/etc/loki/local-config.yaml"]
    volumes:
      - ./observability/loki-config.yaml:/etc/loki/local-config.yaml:ro
      - loki_data:/loki
    ports:
      - "3100:3100"
    networks:
      socialnet: {}

  promtail:
    image: grafana/promtail:2.9.8
    container_name: promtail
    command: ["-config.file=/etc/promtail/config.yml"]
    volumes:
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./observability/promtail-config.yml:/etc/promtail/config.yml:ro
    networks:
      socialnet: {}

  prometheus:
    image: prom/prometheus:v2.55.1
    container_name: prometheus
    volumes:
      - ./observability/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    command:
      - "--config.file=/etc/prometheus/prometheus.yml"
      - "--storage.tsdb.retention.time=15d"
    ports:
      - "9090:9090"
    networks:
      socialnet: {}

  grafana:
    image: grafana/grafana:11.1.0
    container_name: grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_DEFAULT_THEME=light
    volumes:
      - ./observability/grafana-datasources.yaml:/etc/grafana/provisioning/datasources/datasources.yaml
      - ./observability/grafana-dashboards.yaml:/etc/grafana/provisioning/dashboards/dashboards.yaml
      - ./observability/dashboards:/var/lib/grafana/dashboards
    depends_on:
      - prometheus
      - loki
    networks:
      socialnet: {}

  postgres-exporter-shard1:
    image: quay.io/prometheuscommunity/postgres-exporter:v0.15.0
    container_name: postgres-exporter-shard1
    environment:
      DATA_SOURCE_NAME: "postgresql://app:app_pass_shard1@shard1-pgpool:5432/appdb?sslmode=disable"
    ports: ["9187:9187"]
    networks: { socialnet: {} }
    depends_on: { shard1-pgpool: { condition: service_healthy } }

  postgres-exporter-shard2:
    image: quay.io/prometheuscommunity/postgres-exporter:v0.15.0
    container_name: postgres-exporter-shard2
    environment:
      DATA_SOURCE_NAME: "postgresql://app:app_pass_shard2@shard2-pgpool:5432/appdb?sslmode=disable"
    ports: ["9188:9187"]
    networks: { socialnet: {} }
    depends_on: { shard2-pgpool: { condition: service_healthy } }

  redis-exporter-feed:
    image: oliver006/redis_exporter:v1.62.0
    container_name: redis-exporter-feed
    command: ["--redis.addr=redis-feed:6379"]
    ports: ["9121:9121"]
    networks: { socialnet: {} }
    depends_on: { redis-feed: { condition: service_healthy } }

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

  # Observability
  loki_data:
  prometheus_data:
  grafana_data:

here my user service completed now
services/user-service/cmd/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"users-service/internal/user"
	"users-service/pkg/db"

	// Prometheus metrics endpoint
	"github.com/prometheus/client_golang/prometheus/promhttp"

	// OpenTelemetry core
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	// HTTP middleware instrumentation
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	// GORM tracing plugin
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

type ShardPicker interface {
	Pick(shardID int) *gorm.DB
	ForcePrimary(shardID int) *gorm.DB
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func initTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	endpoint := env("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4318")

	exp, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otlptracehttp: %w", err)
	}

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			attribute.String("service.version", "1.0.0"),
			attribute.String("deployment.environment", env("ENV", "local")),
		),
	)

	ratio := 1.0
	if v := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
			ratio = f
		}
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(ratio))),
		trace.WithBatcher(exp,
			trace.WithMaxExportBatchSize(512),
			trace.WithBatchTimeout(3*time.Second),
		),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Honor W3C TraceContext + Baggage for inbound/outbound requests.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		n = 1
	}
	return n
}

func main() {
	ctx := context.Background()

	shutdown, err := initTracer(ctx, "user-service")
	if err != nil {
		log.Fatalf("otel init failed: %v", err)
	}
	// Give exporter time to flush on stop.
	defer func() {
		c, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = shutdown(c)
	}()

	store := db.OpenFromEnv()

	// Enable SQL spans from GORM.
	if err := store.Base.Use(tracing.NewPlugin()); err != nil {
		log.Fatalf("gorm otel plugin failed: %v", err)
	}

	// Auto-migrate across shards if requested.
	if os.Getenv("AUTO_MIGRATE") == "true" {
		numShards := mustAtoi(os.Getenv("NUM_SHARDS"))
		for i := 0; i < numShards; i++ {
			if err := store.ForcePrimary(i).AutoMigrate(&user.User{}); err != nil {
				log.Fatalf("migration failed on shard %d: %v", i, err)
			}
		}
	}

	repo := user.NewUserRepository(store)
	svc := user.NewUserService(repo)
	handler := user.NewUserHandler(svc)

	// App routes.
	api := http.NewServeMux()
	user.RegisterRoutes(api, handler)

	// Root mux: /metrics + OTel-instrumented app.
	root := http.NewServeMux()
	root.Handle("/metrics", promhttp.Handler())
	root.Handle("/", otelhttp.NewHandler(api, "http.server"))

	addr := env("APP_PORT", ":8081")
	srv := &http.Server{
		Addr:              addr,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	fmt.Printf("User Service listening on %s\n", addr)
	log.Fatal(srv.ListenAndServe())
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

services/user-service/pkg/db/db.go
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

services/user-service/pkg/req/decode.go
package req

import (
	"encoding/json"
	"io"
)

func Decode[T any](body io.ReadCloser) (T, error) {
	var payload T
	err := json.NewDecoder(body).Decode(&payload)
	return payload, err
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
		res.Json(*w, map[string]any{"error": "invalid JSON"}, http.StatusBadRequest)
		return nil, err
	}
	if err = IsValid(body); err != nil {
		res.Json(*w, map[string]any{"error": err.Error()}, http.StatusUnprocessableEntity)
		return nil, err
	}
	return &body, nil
}

services/user-service/pkg/req/validate.go
package req

import "github.com/go-playground/validator/v10"

func IsValid[T any](payload T) error {
	validate := validator.New()
	return validate.Struct(payload)
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
	_ = json.NewEncoder(w).Encode(data)
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

services/user-service/internal/user/handler.go
package user

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"users-service/internal/auth"
	"users-service/pkg/req"
	"users-service/pkg/res"
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
			h.ListMine(w, r) // lists only the caller's shard via JWT
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
	body, err := req.HandleBody[RegisterRequest](&w, r)
	if err != nil {
		// HandleBody already wrote 400/422 JSON
		return
	}
	u, err := h.Service.Register(body.Email, body.Password, body.Name)
	if err != nil {
		res.Json(w, map[string]any{"error": err.Error()}, http.StatusConflict)
		return
	}

	// issue a JWT on register as well
	tok, _ := auth.MakeJWT(u.UserID, u.ShardID)

	w.Header().Set("X-Shard-ID", strconv.Itoa(u.ShardID)) // debug only
	res.Json(w, map[string]any{
		"user_id":      u.UserID,
		"email":        u.Email,
		"name":         u.Name,
		"access_token": tok,
	}, http.StatusCreated)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	body, err := req.HandleBody[LoginRequest](&w, r)
	if err != nil {
		return
	}
	u, err := h.Service.Login(body.Email, body.Password)
	if err != nil {
		res.Json(w, map[string]any{"error": "unauthorized"}, http.StatusUnauthorized)
		return
	}
	tok, _ := auth.MakeJWT(u.UserID, u.ShardID)

	w.Header().Set("X-Shard-ID", strconv.Itoa(u.ShardID)) // debug only
	res.Json(w, map[string]any{
		"message":      "login successful",
		"user_id":      u.UserID,
		"name":         u.Name,
		"email":        u.Email,
		"access_token": tok,
	}, http.StatusOK)
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
	Password string `gorm:"size:255" json:"-"` // bcrypt hash
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
	Name    string `json:"name"`
}

services/user-service/internal/user/repository.go
package user

import (
	"errors"
	"log"

	"users-service/pkg/db"
	"users-service/pkg/shard"

	"gorm.io/gorm"
)

type ShardPicker interface {
	Pick(shardID int) *gorm.DB
	ForcePrimary(shardID int) *gorm.DB
	// For logging (optional)
	ShardInfo(shardID int) (db.ShardCfg, bool)
}

type IUserRepository interface {
	Create(u *User) (*User, error)
	FindByEmail(email string, shardID int) (*User, error)
	FindByUserID(uid string) (*User, error)
	FindAllByShard(shardID int) ([]User, error)
	FindAllByShardPaged(shardID, limit, offset int) ([]User, error)
}

type UserRepository struct {
	db ShardPicker
}

func NewUserRepository(p ShardPicker) IUserRepository {
	return &UserRepository{db: p}
}

func (r *UserRepository) logShard(where, role string, shardID int) {
	if cfg, ok := r.db.ShardInfo(shardID); ok {
		w := db.RedactDSN(cfg.Writer)
		var readers string
		if len(cfg.Readers) > 0 {
			readers = db.RedactDSN(cfg.Readers[0])
			if len(cfg.Readers) > 1 {
				readers += " (+more)"
			}
		}
		log.Printf("[repo:%s] role=%s shard=%d writer=[%s] reader0=[%s]", where, role, shardID, w, readers)
	} else {
		log.Printf("[repo:%s] role=%s shard=%d", where, role, shardID)
	}
}

func (r *UserRepository) Create(u *User) (*User, error) {
	r.logShard("Create", "primary", u.ShardID)
	if err := r.db.ForcePrimary(u.ShardID).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(email string, shardID int) (*User, error) {
	r.logShard("FindByEmail", "replica", shardID)
	var u User
	if err := r.db.Pick(shardID).Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByUserID(uid string) (*User, error) {
	sh, ok := shard.ExtractShard(uid)
	if !ok {
		return nil, errors.New("invalid user_id format")
	}
	r.logShard("FindByUserID", "replica", sh)
	var u User
	if err := r.db.Pick(sh).Where("user_id = ?", uid).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindAllByShard(shardID int) ([]User, error) {
	r.logShard("FindAllByShard", "replica", shardID)
	var users []User
	if err := r.db.Pick(shardID).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) FindAllByShardPaged(shardID, limit, offset int) ([]User, error) {
	r.logShard("FindAllByShardPaged", "replica", shardID)
	var users []User
	if err := r.db.Pick(shardID).
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
	"log"
	"os"
	"strconv"

	"users-service/pkg/shard"

	"golang.org/x/crypto/bcrypt"
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
	log.Printf("[user-service] Register route -> picked shard=%d for email=%s", sh, email)

	// ensure uniqueness on the owning shard
	if existing, _ := s.repo.FindByEmail(email, sh); existing != nil {
		return nil, errors.New("user already exists")
	}

	// user_id format: "<shard>-<random64hex>"
	var b [8]byte
	_, _ = rand.Read(b[:])
	uid := fmt.Sprintf("%d-%x", sh, binary.BigEndian.Uint64(b[:]))

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	u := &User{
		UserID:   uid,
		ShardID:  sh,
		Email:    email,
		Password: string(hash),
		Name:     name,
	}
	return s.repo.Create(u)
}

func (s *UserService) Login(email, password string) (*User, error) {
	sh := shard.PickShard(email, s.numShards)
	log.Printf("[user-service] Login route -> picked shard=%d for email=%s", sh, email)

	usr, err := s.repo.FindByEmail(email, sh)
	if err != nil {
		return nil, errors.New("wrong credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(password)) != nil {
		return nil, errors.New("wrong credentials")
	}
	return usr, nil
}

func (s *UserService) ListAll(shardID int) ([]User, error) {
	return s.repo.FindAllByShard(shardID)
}

func (s *UserService) GetByUserID(uid string) (*User, error) {
	sh, ok := shard.ExtractShard(uid)
	if ok {
		log.Printf("[user-service] GetByUserID -> extracted shard=%d from user_id=%s", sh, uid)
	}
	return s.repo.FindByUserID(uid)
}

func (s *UserService) ListShard(shardID, limit, offset int) ([]User, error) {
	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	log.Printf("[user-service] ListShard -> shard=%d limit=%d offset=%d", shardID, limit, offset)
	return s.repo.FindAllByShardPaged(shardID, limit, offset)
}

services/user-service/internal/auth/jwt.go
package auth

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func secret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		// dev fallback only; set JWT_SECRET in production
		s = "replace-this-with-a-strong-secret"
	}
	return []byte(s)
}

func MakeJWT(userID string, shardID int) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"sh":  shardID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret())
}

func ParseAuthHeader(authz string) (userID string, shardID int, err error) {
	if authz == "" {
		return "", 0, errors.New("missing Authorization")
	}
	tokenStr := strings.TrimPrefix(authz, "Bearer ")
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return secret(), nil
	})
	if err != nil || !tok.Valid {
		return "", 0, errors.New("invalid token")
	}
	mc, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", 0, errors.New("bad claims")
	}
	uid, _ := mc["sub"].(string)
	shf, ok := mc["sh"].(float64)
	if !ok {
		return "", 0, errors.New("missing shard claim")
	}
	return uid, int(shf), nil
}