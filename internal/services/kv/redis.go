package kv

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/the127/dockyard/internal/config"
)

func NewRedisStore(kvConfig config.KvConfig) Store {
	return &redisKvStore{
		kvConfig: kvConfig,
	}
}

type redisKvStore struct {
	kvConfig config.KvConfig
}

func (r *redisKvStore) Set(ctx context.Context, key string, value string, opts ...Option) error {
	client := r.newRedisClient()
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return client.Set(ctx, key, value, options.Expiration).Err()
}

func (r *redisKvStore) Get(ctx context.Context, key string) (string, bool, error) {
	client := r.newRedisClient()
	result, err := client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	return result, true, err
}

func (r *redisKvStore) Delete(ctx context.Context, key string) error {
	client := r.newRedisClient()
	err := client.Del(ctx, key).Err()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	return err
}

func (r *redisKvStore) newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", r.kvConfig.Redis.Host, r.kvConfig.Redis.Port),
		Username: r.kvConfig.Redis.Username,
		Password: r.kvConfig.Redis.Password,
		DB:       r.kvConfig.Redis.Database,
	})
}
