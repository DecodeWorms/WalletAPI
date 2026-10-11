package storage

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type CacheStorage interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
}

type CacStorage struct {
	store *redis.Client
}

func NewCacStorage(c *redis.Client) CacStorage {
	return CacStorage{
		store: c,
	}
}

func (c CacStorage) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.store.Set(ctx, key, value, 30*ttl).Err()
}

func (c CacStorage) Get(ctx context.Context, key string) (string, error) {
	return c.store.Get(ctx, key).Result()
}

func (c CacStorage) Del(ctx context.Context, key string) error {
	return c.store.Del(ctx, key).Err()
}

func (c CacStorage) Incr(ctx context.Context, key string) (int64, error) {
	return c.store.Incr(ctx, key).Result()
}

func (c CacStorage) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.store.Expire(ctx, key, ttl).Err()
}

func (c CacStorage) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.store.SetNX(ctx, key, value, ttl).Result()
}
