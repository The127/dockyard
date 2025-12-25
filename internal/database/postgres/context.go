package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/the127/dockyard/internal/change"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/repositories/postgres"
	"github.com/the127/dockyard/internal/utils"
)

type Context struct {
	db            *sql.DB
	changeTracker *change.Tracker

	tenants          *postgres.TenantRepository
	projects         *postgres.ProjectRepository
	projectAccess    *postgres.ProjectAccessRepository
	users            *postgres.UserRepository
	pats             *postgres.PatRepository
	repos            *postgres.RepositoryRepository
	repositoryAccess *postgres.RepositoryAccessRepository
	manifest         *postgres.ManifestRepository
	tags             *postgres.TagRepository
	blobs            *postgres.BlobRepository
	repositoryBlobs  *postgres.RepositoryBlobRepository
	files            *postgres.FileRepository
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
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: 0,
		ReadOnly:  false,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer utils.IgnoreError(tx.Rollback)

	changes := c.changeTracker.GetChanges()
	for _, changeEntry := range changes {
		err := c.applyChange(ctx, tx, changeEntry)
		if err != nil {
			return fmt.Errorf("failed to apply change: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (c *Context) applyChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetItemType() {
	case db.TenantType:
		return c.applyTenantChange(ctx, tx, entry)

	case db.ProjectType:
		return c.applyProjectChange(ctx, tx, entry)

	case db.ProjectAccessType:
		return c.applyProjectAccessChange(ctx, tx, entry)

	case db.UserType:
		return c.applyUserChange(ctx, tx, entry)

	case db.PatType:
		return c.applyPatChange(ctx, tx, entry)

	case db.RepositoryType:
		return c.applyRepositoryChange(ctx, tx, entry)

	case db.RepositoryAccessType:
		return c.applyRepositoryAccessChange(ctx, tx, entry)

	case db.ManifestType:
		return c.applyManifestChange(ctx, tx, entry)

	case db.TagType:
		return c.applyTagChange(ctx, tx, entry)

	case db.BlobType:
		return c.applyBlobChange(ctx, tx, entry)

	case db.RepositoryBlobType:
		return c.applyRepositoryBlobChange(ctx, tx, entry)

	case db.FileType:
		return c.applyFileChange(ctx, tx, entry)

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyTenantChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.tenants.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.Tenant))

	case change.Updated:
		return c.tenants.ExecuteUpdate(ctx, tx, entry.GetItem().(*repositories.Tenant))

	case change.Deleted:
		return c.tenants.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.Tenant))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyProjectChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.projects.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.Project))

	case change.Updated:
		return c.projects.ExecuteUpdate(ctx, tx, entry.GetItem().(*repositories.Project))

	case change.Deleted:
		return c.projects.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.Project))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyProjectAccessChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.projectAccess.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.ProjectAccess))

	case change.Updated:
		return c.projectAccess.ExecuteUpdate(ctx, tx, entry.GetItem().(*repositories.ProjectAccess))

	case change.Deleted:
		return c.projectAccess.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.ProjectAccess))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyUserChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.users.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.User))

	case change.Updated:
		return c.users.ExecuteUpdate(ctx, tx, entry.GetItem().(*repositories.User))

	case change.Deleted:
		return c.users.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.User))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyPatChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.pats.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.Pat))

	case change.Updated:
		return c.pats.ExecuteUpdate(ctx, tx, entry.GetItem().(*repositories.Pat))

	case change.Deleted:
		return c.pats.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.Pat))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyRepositoryChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.repos.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.Repository))

	case change.Updated:
		return c.repos.ExecuteUpdate(ctx, tx, entry.GetItem().(*repositories.Repository))

	case change.Deleted:
		return c.repos.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.Repository))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyRepositoryAccessChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.repositoryAccess.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.RepositoryAccess))

	case change.Updated:
		return c.repositoryAccess.ExecuteUpdate(ctx, tx, entry.GetItem().(*repositories.RepositoryAccess))

	case change.Deleted:
		return c.repositoryAccess.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.RepositoryAccess))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyManifestChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.manifest.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.Manifest))

	case change.Deleted:
		return c.manifest.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.Manifest))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyTagChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.tags.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.Tag))

	case change.Deleted:
		return c.tags.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.Tag))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyBlobChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.blobs.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.Blob))

	case change.Deleted:
		return c.blobs.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.Blob))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyRepositoryBlobChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.repositoryBlobs.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.RepositoryBlob))

	case change.Deleted:
		return c.repositoryBlobs.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.RepositoryBlob))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}

func (c *Context) applyFileChange(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.files.ExecuteInsert(ctx, tx, entry.GetItem().(*repositories.File))

	case change.Deleted:
		return c.files.ExecuteDelete(ctx, tx, entry.GetItem().(*repositories.File))

	default:
		return fmt.Errorf("unsupported change type: %s", entry.GetChangeType())
	}
}
