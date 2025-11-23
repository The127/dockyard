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
	if t.tenants == nil {
		t.tenants = postgres.NewPostgresTenantRepository(t.tx)
	}

	return t.tenants
}

func (t *transaction) Projects() repositories.ProjectRepository {
	if t.projects == nil {
		t.projects = postgres.NewPostgresProjectRepository(t.tx)
	}

	return t.projects
}

func (t *transaction) ProjectAccess() repositories.ProjectAccessRepository {
	if t.projectAccess == nil {
		t.projectAccess = postgres.NewPostgresProjectAccessRepository(t.tx)
	}

	return t.projectAccess
}

func (t *transaction) Users() repositories.UserRepository {
	if t.users == nil {
		t.users = postgres.NewPostgresUserRepository(t.tx)
	}

	return t.users
}

func (t *transaction) Pats() repositories.PatRepository {
	if t.pats == nil {
		t.pats = postgres.NewPostgresPatRepository(t.tx)
	}

	return t.pats
}

func (t *transaction) Repositories() repositories.RepositoryRepository {
	if t.repos == nil {
		t.repos = postgres.NewPostgresRepositoryRepository(t.tx)
	}

	return t.repos
}

func (t *transaction) RepositoryAccess() repositories.RepositoryAccessRepository {
	if t.repositoryAccess == nil {
		t.repositoryAccess = postgres.NewPostgresRepositoryAccessRepository(t.tx)
	}

	return t.repositoryAccess
}

func (t *transaction) Manifests() repositories.ManifestRepository {
	if t.manifest == nil {
		t.manifest = postgres.NewPostgresManifestRepository(t.tx)
	}

	return t.manifest
}

func (t *transaction) Tags() repositories.TagRepository {
	if t.tags == nil {
		t.tags = postgres.NewPostgresTagRepository(t.tx)
	}

	return t.tags
}

func (t *transaction) Blobs() repositories.BlobRepository {
	if t.blobs == nil {
		t.blobs = postgres.NewPostgresBlobRepository(t.tx)
	}

	return t.blobs
}

func (t *transaction) RepositoryBlobs() repositories.RepositoryBlobRepository {
	if t.repositoryBlobs == nil {
		t.repositoryBlobs = postgres.NewPostgresRepositoryBlobRepository(t.tx)
	}

	return t.repositoryBlobs
}

func (t *transaction) Files() repositories.FileRepository {
	if t.files == nil {
		t.files = postgres.NewPostgresFileRepository(t.tx)
	}

	return t.files
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
