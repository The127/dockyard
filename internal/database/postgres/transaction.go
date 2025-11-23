package postgres

import (
	"database/sql"

	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/repositories/postgres"
)

type transaction struct {
	tx *sql.Tx

	tenants          repositories.TenantRepository
	projects         repositories.ProjectRepository
	projectAccess    repositories.ProjectAccessRepository
	users            repositories.UserRepository
	pats             repositories.PatRepository
	repos            repositories.RepositoryRepository
	repositoryAccess repositories.RepositoryAccessRepository
	manifest         repositories.ManifestRepository
	tags             repositories.TagRepository
	blobs            repositories.BlobRepository
	repositoryBlobs  repositories.RepositoryBlobRepository
	files            repositories.FileRepository
}

func (t *transaction) Tenants() repositories.TenantRepository {
	//TODO implement me
	panic("implement me")
}

func (t *transaction) Projects() repositories.ProjectRepository {
	//TODO implement me
	panic("implement me")
}

func (t *transaction) ProjectAccess() repositories.ProjectAccessRepository {
	//TODO implement me
	panic("implement me")
}

func (t *transaction) Users() repositories.UserRepository {
	//TODO implement me
	panic("implement me")
}

func (t *transaction) Pats() repositories.PatRepository {
	//TODO implement me
	panic("implement me")
}

func (t *transaction) Repositories() repositories.RepositoryRepository {
	//TODO implement me
	panic("implement me")
}

func (t *transaction) RepositoryAccess() repositories.RepositoryAccessRepository {
	//TODO implement me
	panic("implement me")
}

func (t *transaction) Manifests() repositories.ManifestRepository {
	//TODO implement me
	panic("implement me")
}

func (t *transaction) Tags() repositories.TagRepository {
	//TODO implement me
	panic("implement me")
}

func (t *transaction) Blobs() repositories.BlobRepository {
	if t.blobs == nil {
		t.blobs = postgres.NewPostgresBlobRepository(t.tx)
	}

	return t.blobs
}

func (t *transaction) RepositoryBlobs() repositories.RepositoryBlobRepository {
	//TODO implement me
	panic("implement me")
}

func (t *transaction) Files() repositories.FileRepository {
	//TODO implement me
	panic("implement me")
}

func newTransaction(tx *sql.Tx) db.Transaction {
	return &transaction{
		tx: tx,
	}
}

func (t *transaction) Commit() error {
	return nil
}

func (t *transaction) Rollback() error {
	return nil
}
