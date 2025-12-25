package database

import "context"

type Database interface {
	Migrate() error
	NewContext(ctx context.Context) (Context, error)
}
