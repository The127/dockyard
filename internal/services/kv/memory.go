package kv

import (
	"context"
	"time"

	"github.com/patrickmn/go-cache"
)

func NewMemoryStore() Store {
	return &memoryStore{
		cache: cache.New(-1, 10*time.Minute),
	}
}

type memoryStore struct {
	cache *cache.Cache
}

func (m *memoryStore) Get(ctx context.Context, key string) (value string, ok bool, error error) {
	result, ok := m.cache.Get(key)
	return result.(string), ok, nil
}

func (m *memoryStore) Set(ctx context.Context, key string, value string, opts ...Option) error {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	m.cache.Set(key, value, options.Expiration)
	return nil
}

func (m *memoryStore) Delete(ctx context.Context, key string) error {
	m.cache.Delete(key)
	return nil
}
