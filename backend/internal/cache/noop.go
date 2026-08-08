package cache

import (
	"context"
	"time"
)

// NoopCache is a no-op Cache used when Redis is unavailable, so the backend
// keeps working with caching disabled.
type NoopCache struct{}

func (NoopCache) Get(ctx context.Context, key string) (string, bool) { return "", false }
func (NoopCache) Set(ctx context.Context, key, value string, ttl time.Duration) {
}
func (NoopCache) Del(ctx context.Context, key string)        {}
func (NoopCache) Flush(ctx context.Context)                  {}
