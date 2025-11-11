package kv

import (
	"context"
	"time"
)

type Options struct {
	Expiration time.Duration
}

type Option func(*Options)

func WithExpiration(expiration time.Duration) Option {
	return func(o *Options) {
		o.Expiration = expiration
	}
}

type Store interface {
	Get(ctx context.Context, key string) (value string, ok bool, error error)
	Set(ctx context.Context, key string, value string, opts ...Option) error
	Delete(ctx context.Context, key string) error
}
