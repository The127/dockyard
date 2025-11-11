package inmemory

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/repositories"
)

type database struct {
	memDB *memdb.MemDB
}

func NewInMemoryDatabase() (db.Database, error) {
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
						Indexer: &UUIDValueIndexer{Getter: func(obj interface{}) uuid.UUID {
							repositoryBlob := obj.(repositories.RepositoryBlob)
							return repositoryBlob.GetId()
						}},
					},
				},
			},
		},
	}

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
	return nil
}

func (d *database) Tx() (db.Transaction, error) {
	return newTransaction(d.memDB.Txn(true)), nil
}
