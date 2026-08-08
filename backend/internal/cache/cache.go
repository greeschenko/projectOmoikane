package cache

import (
	"context"
	"time"
)

// Cache is the interface used for HTTP response caching.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key string, value string, ttl time.Duration)
	Del(ctx context.Context, key string)
	Flush(ctx context.Context)
}
