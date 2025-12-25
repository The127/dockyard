package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/the127/dockyard/internal/change"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/repositories/postgres"
)

type Context struct {
	db            *sql.DB
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

func newContext(db *sql.DB) *Context {
	return &Context{
		db:            db,
		changeTracker: change.NewTracker(),
	}
}

func (c *Context) Tenants() repositories.TenantRepository {
	if c.tenants == nil {
		c.tenants = postgres.NewPostgresTenantRepository(c.db, c.changeTracker, db.TenantType)
	}

	return c.tenants
}

func (c *Context) Projects() repositories.ProjectRepository {
	if c.projects == nil {
		c.projects = postgres.NewPostgresProjectRepository(c.db, c.changeTracker, db.ProjectType)
	}

	return c.projects
}

func (c *Context) ProjectAccess() repositories.ProjectAccessRepository {
	if c.projectAccess == nil {
		c.projectAccess = postgres.NewPostgresProjectAccessRepository(c.db, c.changeTracker, db.ProjectAccessType)
	}

	return c.projectAccess
}

func (c *Context) Users() repositories.UserRepository {
	if c.users == nil {
		c.users = postgres.NewPostgresUserRepository(c.db, c.changeTracker, db.UserType)
	}

	return c.users
}

func (c *Context) Pats() repositories.PatRepository {
	if c.pats == nil {
		c.pats = postgres.NewPostgresPatRepository(c.db, c.changeTracker, db.PatType)
	}

	return c.pats
}

func (c *Context) Repositories() repositories.RepositoryRepository {
	if c.repos == nil {
		c.repos = postgres.NewPostgresRepositoryRepository(c.db, c.changeTracker, db.RepositoryType)
	}

	return c.repos
}

func (c *Context) RepositoryAccess() repositories.RepositoryAccessRepository {
	if c.repositoryAccess == nil {
		c.repositoryAccess = postgres.NewPostgresRepositoryAccessRepository(c.db, c.changeTracker, db.RepositoryAccessType)
	}

	return c.repositoryAccess
}

func (c *Context) Manifests() repositories.ManifestRepository {
	if c.manifest == nil {
		c.manifest = postgres.NewPostgresManifestRepository(c.db, c.changeTracker, db.ManifestType)
	}

	return c.manifest
}

func (c *Context) Tags() repositories.TagRepository {
	if c.tags == nil {
		c.tags = postgres.NewPostgresTagRepository(c.db, c.changeTracker, db.TagType)
	}

	return c.tags
}

func (c *Context) Blobs() repositories.BlobRepository {
	if c.blobs == nil {
		c.blobs = postgres.NewPostgresBlobRepository(c.db, c.changeTracker, db.BlobType)
	}

	return c.blobs
}

func (c *Context) RepositoryBlobs() repositories.RepositoryBlobRepository {
	if c.repositoryBlobs == nil {
		c.repositoryBlobs = postgres.NewPostgresRepositoryBlobRepository(c.db, c.changeTracker, db.RepositoryBlobType)
	}

	return c.repositoryBlobs
}

func (c *Context) Files() repositories.FileRepository {
	if c.files == nil {
		c.files = postgres.NewPostgresFileRepository(c.db, c.changeTracker, db.FileType)
	}

	return c.files
}

func (c *Context) SaveChanges(ctx context.Context) error {
	// TODO: implement
	return errors.New(fmt.Sprintf("not implemented"))
}
