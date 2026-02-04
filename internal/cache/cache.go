package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, value string, ttl time.Duration)
}

type RedisCache struct {
	client *redis.Client
}

func NewRedis(addr string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &RedisCache{client: client}, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, bool) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

func (r *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) {
	_ = r.client.Set(ctx, key, value, ttl).Err()
}
