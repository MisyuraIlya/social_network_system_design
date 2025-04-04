package redis

import (
	"log"
	"time"

	"github.com/gomodule/redigo/redis"

	"feed-service/configs"
)

type RedisPool struct {
	Pool *redis.Pool
}

func NewRedisPool(cfg *configs.Config) *RedisPool {
	pool := &redis.Pool{
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cfg.RedisAddr())
			if err != nil {
				return nil, err
			}
			// Auth if password is provided
			if cfg.RedisPass != "" {
				if _, err := c.Do("AUTH", cfg.RedisPass); err != nil {
					_ = c.Close()
					return nil, err
				}
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return &RedisPool{Pool: pool}
}

func (rp *RedisPool) GetConn() redis.Conn {
	conn := rp.Pool.Get()
	if err := conn.Err(); err != nil {
		log.Printf("Redis connection error: %v", err)
	}
	return conn
}
