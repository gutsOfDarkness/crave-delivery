// Package redis provides Redis client with connection management.
// Used for session management and menu caching.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"fooddelivery/pkg/logger"
)

// Client wraps redis.Client with additional functionality
type Client struct {
	*redis.Client
	log *logger.Logger
}

// NewClient creates a new Redis client with the given connection URL.
// URL format: redis://:password@host:port/db
func NewClient(url string, log *logger.Logger) (*Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Connection pool settings optimized for caching workload
	opts.PoolSize = 20
	opts.MinIdleConns = 5
	opts.MaxRetries = 3
	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second
	opts.PoolTimeout = 4 * time.Second

	client := redis.NewClient(opts)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info("Redis connection established")

	return &Client{
		Client: client,
		log:    log,
	}, nil
}

// Cache keys constants
const (
	MenuCacheKey       = "app:menu:all"
	MenuCacheTTL       = 1 * time.Hour
	IdempotencyPrefix  = "app:idempotency:"
	IdempotencyTTL     = 1 * time.Minute
	SessionPrefix      = "app:session:"
	SessionTTL         = 24 * time.Hour
)

// GetJSON retrieves a JSON value from Redis and unmarshals it into the target.
// Returns false if key doesn't exist.
func (c *Client) GetJSON(ctx context.Context, key string, target interface{}) (bool, error) {
	val, err := c.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil // Cache miss
	}
	if err != nil {
		return false, fmt.Errorf("redis get failed: %w", err)
	}

	if err := json.Unmarshal([]byte(val), target); err != nil {
		return false, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return true, nil
}

// SetJSON marshals the value to JSON and stores it in Redis with TTL.
func (c *Client) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := c.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

// DeleteKey removes a key from Redis.
// Used for cache invalidation.
func (c *Client) DeleteKey(ctx context.Context, key string) error {
	if err := c.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}
	return nil
}

// SetNXWithTTL sets a key only if it doesn't exist (for idempotency).
// Returns true if the key was set (first request), false if it already exists.
// This is the foundation for preventing duplicate order creation.
func (c *Client) SetNXWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	// SetNX is atomic - only one concurrent request will succeed
	result, err := c.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx failed: %w", err)
	}

	return result, nil
}

// GetAndExtendTTL retrieves a value and extends its TTL.
// Useful for session management where activity should extend session life.
func (c *Client) GetAndExtendTTL(ctx context.Context, key string, target interface{}, newTTL time.Duration) (bool, error) {
	// Use pipeline for atomic get + expire
	pipe := c.Pipeline()
	getCmd := pipe.Get(ctx, key)
	pipe.Expire(ctx, key, newTTL)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("redis pipeline failed: %w", err)
	}

	val, err := getCmd.Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get failed: %w", err)
	}

	if err := json.Unmarshal([]byte(val), target); err != nil {
		return false, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return true, nil
}
