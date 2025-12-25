package database

import (
	"context"
)

type Factory interface {
	NewDbContext(ctx context.Context) (Context, error)
}

type factory struct {
	db Database
}

func NewDbFactory(db Database) Factory {
	return &factory{
		db: db,
	}
}

func (s *factory) NewDbContext(ctx context.Context) (Context, error) {
	return s.db.Tx(ctx)
}
