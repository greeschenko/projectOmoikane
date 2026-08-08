package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedis connects to Redis and returns a Cache. An error is returned if the
// connection fails, so callers can fall back to a noop cache.
func NewRedis(url string, ttl time.Duration) (Cache, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	log.Println("Redis cache connected")
	return &RedisCache{client: client, ttl: ttl}, nil
}

// NewRedisClient wraps an existing go-redis client (used in tests to simulate
// an unreachable Redis).
func NewRedisClient(client *redis.Client) Cache {
	return &RedisCache{client: client, ttl: time.Minute}
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, bool) {
	v, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return v, true
}

func (c *RedisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) {
	if ttl <= 0 {
		ttl = c.ttl
	}
	c.client.Set(ctx, key, value, ttl)
}

func (c *RedisCache) Del(ctx context.Context, key string) {
	c.client.Del(ctx, key)
}

func (c *RedisCache) Flush(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.client.FlushDB(ctx)
}
