package cache

import (
	"context"
	"delayedNotifier/internal/application/contracts"
	"time"

	"github.com/wb-go/wbf/redis"
)

type RedisCache struct {
	client  *redis.Client
	retryer contracts.Retryer
}

func NewRedisCache(client *redis.Client, retryer contracts.Retryer) *RedisCache {
	return &RedisCache{
		client:  client,
		retryer: retryer,
	}
}

// Set zero expiry means the key has no expiration time
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
	return c.retryer.Retry(func() error {
		return c.client.SetWithExpiration(ctx, key, value, expiry)
	})
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key)
}
