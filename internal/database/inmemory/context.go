package inmemory

import (
	"context"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/repositories/inmemory"
)

type Context struct {
	db            *memdb.MemDB
	txn           *memdb.Txn
	changeTracker *change.Tracker

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

func newContext(db *memdb.MemDB) *Context {
	return &Context{
		db:  db,
		txn: db.Txn(false),
	}
}

func (c *Context) Tenants() repositories.TenantRepository {
	if c.tenants == nil {
		c.tenants = inmemory.NewInMemoryTenantRepository(c.txn, c.changeTracker, db.TenantType)
	}
	return c.tenants
}

func (c *Context) Projects() repositories.ProjectRepository {
	if c.projects == nil {
		c.projects = inmemory.NewInMemoryProjectRepository(c.txn, c.changeTracker, db.ProjectType)
	}
	return c.projects
}

func (c *Context) ProjectAccess() repositories.ProjectAccessRepository {
	if c.projectAccess == nil {
		c.projectAccess = inmemory.NewInMemoryProjectAccessRepository(c.txn, c.changeTracker, db.ProjectAccessType)
	}
	return c.projectAccess
}

func (c *Context) Users() repositories.UserRepository {
	if c.users == nil {
		c.users = inmemory.NewInMemoryUserRepository(c.txn, c.changeTracker, db.UserType)
	}
	return c.users
}

func (c *Context) Pats() repositories.PatRepository {
	if c.pats == nil {
		c.pats = inmemory.NewInMemoryPatRepository(c.txn, c.changeTracker, db.PatType)
	}
	return c.pats
}

func (c *Context) Repositories() repositories.RepositoryRepository {
	if c.repos == nil {
		c.repos = inmemory.NewInMemoryRepositoryRepository(c.txn, c.changeTracker, db.RepositoryType)
	}
	return c.repos
}

func (c *Context) RepositoryAccess() repositories.RepositoryAccessRepository {
	if c.repositoryAccess == nil {
		c.repositoryAccess = inmemory.NewInMemoryRepositoryAccessRepository(c.txn, c.changeTracker, db.RepositoryAccessType)
	}
	return c.repositoryAccess
}

func (c *Context) Manifests() repositories.ManifestRepository {
	if c.manifest == nil {
		c.manifest = inmemory.NewInMemoryManifestRepository(c.txn, c.changeTracker, db.ManifestType)
	}
	return c.manifest
}

func (c *Context) Blobs() repositories.BlobRepository {
	if c.blobs == nil {
		c.blobs = inmemory.NewInMemoryBlobRepository(c.txn, c.changeTracker, db.BlobType)
	}
	return c.blobs
}

func (c *Context) RepositoryBlobs() repositories.RepositoryBlobRepository {
	if c.repositoryBlobs == nil {
		c.repositoryBlobs = inmemory.NewInMemoryRepositoryBlobRepository(c.txn, c.changeTracker, db.RepositoryBlobType)
	}
	return c.repositoryBlobs
}

func (c *Context) Tags() repositories.TagRepository {
	if c.tags == nil {
		c.tags = inmemory.NewInMemoryTagRepository(c.txn, c.changeTracker, db.TagType)
	}
	return c.tags
}

func (c *Context) Files() repositories.FileRepository {
	if c.files == nil {
		c.files = inmemory.NewInMemoryFileRepository(c.txn, c.changeTracker, db.FileType)
	}
	return c.files
}

func (c *Context) SaveChanges(ctx context.Context) error {
	_ = c.db.Txn(true)
	// TODO: implement
	return nil
}
