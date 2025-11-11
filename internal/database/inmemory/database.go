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

var schema *memdb.DBSchema

func (d *database) Migrate() error {
	schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"tenants": {
				Name: "tenants",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "slug"},
					},
				},
			},
			"projects": {
				Name: "projects",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "slug"},
					},
				},
			},
			"repositories": {
				Name: "repositories",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "slug"},
					},
				},
			},
			"blobs": {
				Name: "blobs",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "digest"},
					},
				},
			},
			"repository_blobs": {
				Name: "repository_blobs",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:   "id",
						Unique: true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.UUIDFieldIndex{Field: "repositoryId"},
								&memdb.UUIDFieldIndex{Field: "blobId"},
							},
						},
					},
				},
			},
		},
	}

	return nil
}

func (d *database) Tx() (db.Transaction, error) {
	return newTransaction(d.memDB.Txn(true)), nil
}
