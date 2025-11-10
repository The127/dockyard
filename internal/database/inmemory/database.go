package inmemory

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
	db "github.com/the127/dockyard/internal/database"
)

type database struct {
	memDB *memdb.MemDB
}

func NewInMemoryDatabase() (db.Database, error) {
	memDb, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create in-memory database: %w", err)
	}

	return &database{
		memDB: memDb,
	}, nil
}

var schema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		"tenants": {
			Name: "tenants",
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.UUIDFieldIndex{Field: "id"},
				},
			},
		},
	},
}

func (d *database) Migrate() error {
	return nil
}

func (d *database) Tx() (db.Transaction, error) {
	return newTransaction(d.memDB.Txn(true)), nil
}
