package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(addr, password string, db int) (*RedisClient, error) {
	log.Printf("[Redis] Connecting to Redis at %s (db=%d)", addr, db)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("[Redis] Successfully connected to Redis")
	return &RedisClient{client: client}, nil
}

// IsTokenBlacklisted checks if a token is in the blacklist
// This checks the same Redis keys that User Service uses for logout
func (r *RedisClient) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", token)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		log.Printf("[Redis] Error checking blacklist: token=%s, error=%v", token[:20], err)
		return false, err
	}

	if exists > 0 {
		log.Printf("[Redis] Token is blacklisted: token=%s", token[:20])
		return true, nil
	}

	return false, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if r.client != nil {
		log.Printf("[Redis] Closing Redis connection")
		return r.client.Close()
	}
	return nil
}

// HealthCheck verifies Redis connection is healthy
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
