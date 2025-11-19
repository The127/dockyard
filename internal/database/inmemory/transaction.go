package inmemory

import (
	"github.com/hashicorp/go-memdb"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/repositories/inmemory"
)

type transaction struct {
	txn *memdb.Txn

	tenants         repositories.TenantRepository
	projects        repositories.ProjectRepository
	projectAccess   repositories.ProjectAccessRepository
	users           repositories.UserRepository
	pats            repositories.PatRepository
	repos           repositories.RepositoryRepository
	manifest        repositories.ManifestRepository
	tags            repositories.TagRepository
	blobs           repositories.BlobRepository
	repositoryBlobs repositories.RepositoryBlobRepository
	files           repositories.FileRepository
}

func newTransaction(txn *memdb.Txn) db.Transaction {
	return &transaction{
		txn: txn,
	}
}

func (t *transaction) Tenants() repositories.TenantRepository {
	if t.tenants == nil {
		t.tenants = inmemory.NewInMemoryTenantRepository(t.txn)
	}
	return t.tenants
}

func (t *transaction) Projects() repositories.ProjectRepository {
	if t.projects == nil {
		t.projects = inmemory.NewInMemoryProjectRepository(t.txn)
	}
	return t.projects
}

func (t *transaction) ProjectAccess() repositories.ProjectAccessRepository {
	if t.projectAccess == nil {
		t.projectAccess = inmemory.NewInMemoryProjectAccessRepository(t.txn)
	}
	return t.projectAccess
}

func (t *transaction) Users() repositories.UserRepository {
	if t.users == nil {
		t.users = inmemory.NewInMemoryUserRepository(t.txn)
	}
	return t.users
}

func (t *transaction) Pats() repositories.PatRepository {
	if t.pats == nil {
		t.pats = inmemory.NewInMemoryPatRepository(t.txn)
	}
	return t.pats
}

func (t *transaction) Repositories() repositories.RepositoryRepository {
	if t.repos == nil {
		t.repos = inmemory.NewInMemoryRepositoryRepository(t.txn)
	}
	return t.repos
}

func (t *transaction) Manifests() repositories.ManifestRepository {
	if t.manifest == nil {
		t.manifest = inmemory.NewInMemoryManifestRepository(t.txn)
	}
	return t.manifest
}

func (t *transaction) Blobs() repositories.BlobRepository {
	if t.blobs == nil {
		t.blobs = inmemory.NewInMemoryBlobRepository(t.txn)
	}
	return t.blobs
}

func (t *transaction) RepositoryBlobs() repositories.RepositoryBlobRepository {
	if t.repositoryBlobs == nil {
		t.repositoryBlobs = inmemory.NewInMemoryRepositoryBlobRepository(t.txn)
	}
	return t.repositoryBlobs
}

func (t *transaction) Tags() repositories.TagRepository {
	if t.tags == nil {
		t.tags = inmemory.NewInMemoryTagRepository(t.txn)
	}
	return t.tags
}

func (t *transaction) Files() repositories.FileRepository {
	if t.files == nil {
		t.files = inmemory.NewInMemoryFileRepository(t.txn)
	}
	return t.files
}

func (t *transaction) Commit() error {
	t.txn.Commit()
	return nil
}

func (t *transaction) Rollback() error {
	t.txn.Abort()
	return nil
}
