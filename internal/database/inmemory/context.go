package inmemory

import (
	"context"
	"fmt"

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

	tenants          *inmemory.TenantRepository
	projects         *inmemory.ProjectRepository
	projectAccess    *inmemory.ProjectAccessRepository
	users            *inmemory.UserRepository
	pats             *inmemory.PatRepository
	repos            *inmemory.RepositoryRepository
	repositoryAccess *inmemory.RepositoryAccessRepository
	manifest         *inmemory.ManifestRepository
	tags             *inmemory.TagRepository
	blobs            *inmemory.BlobRepository
	repositoryBlobs  *inmemory.RepositoryBlobRepository
	files            *inmemory.FileRepository
}

func newContext(db *memdb.MemDB) *Context {
	return &Context{
		db:            db,
		txn:           db.Txn(false),
		changeTracker: change.NewTracker(),
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
	tx := c.db.Txn(true)

	changes := c.changeTracker.GetChanges()
	for _, changeEntry := range changes {
		err := c.applyChange(tx, changeEntry)
		if err != nil {
			return fmt.Errorf("failed to apply change: %w", err)
		}
	}

	tx.Commit()
	return nil
}

func (c *Context) applyChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetItemType() {
	case db.TenantType:
		return c.applyTenantChange(tx, entry)

	case db.ProjectType:
		return c.applyProjectChange(tx, entry)

	case db.ProjectAccessType:
		return c.applyProjectAccessChange(tx, entry)

	case db.UserType:
		return c.applyUserChange(tx, entry)

	case db.PatType:
		return c.applyPatChange(tx, entry)

	case db.RepositoryType:
		return c.applyRepositoryChange(tx, entry)

	case db.RepositoryAccessType:
		return c.applyRepositoryAccessChange(tx, entry)

	case db.ManifestType:
		return c.applyManifestChange(tx, entry)

	case db.TagType:
		return c.applyTagChange(tx, entry)

	case db.BlobType:
		return c.applyBlobChange(tx, entry)

	case db.RepositoryBlobType:
		return c.applyRepositoryBlobChange(tx, entry)

	case db.FileType:
		return c.applyFileChange(tx, entry)

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyTenantChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.tenants.ExecuteInsert(tx, entry.GetItem().(*repositories.Tenant))

	case change.Updated:
		return c.tenants.ExecuteUpdate(tx, entry.GetItem().(*repositories.Tenant))

	case change.Deleted:
		return c.tenants.ExecuteDelete(tx, entry.GetItem().(*repositories.Tenant))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyProjectChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.projects.ExecuteInsert(tx, entry.GetItem().(*repositories.Project))

	case change.Updated:
		return c.projects.ExecuteUpdate(tx, entry.GetItem().(*repositories.Project))

	case change.Deleted:
		return c.projects.ExecuteDelete(tx, entry.GetItem().(*repositories.Project))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyProjectAccessChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.projectAccess.ExecuteInsert(tx, entry.GetItem().(*repositories.ProjectAccess))

	case change.Updated:
		return c.projectAccess.ExecuteUpdate(tx, entry.GetItem().(*repositories.ProjectAccess))

	case change.Deleted:
		return c.projectAccess.ExecuteDelete(tx, entry.GetItem().(*repositories.ProjectAccess))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyUserChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.users.ExecuteInsert(tx, entry.GetItem().(*repositories.User))

	case change.Updated:
		return c.users.ExecuteUpdate(tx, entry.GetItem().(*repositories.User))

	case change.Deleted:
		return c.users.ExecuteDelete(tx, entry.GetItem().(*repositories.User))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyPatChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.pats.ExecuteInsert(tx, entry.GetItem().(*repositories.Pat))

	case change.Updated:
		return c.pats.ExecuteUpdate(tx, entry.GetItem().(*repositories.Pat))

	case change.Deleted:
		return c.pats.ExecuteDelete(tx, entry.GetItem().(*repositories.Pat))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyRepositoryChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.repos.ExecuteInsert(tx, entry.GetItem().(*repositories.Repository))

	case change.Updated:
		return c.repos.ExecuteUpdate(tx, entry.GetItem().(*repositories.Repository))

	case change.Deleted:
		return c.repos.ExecuteDelete(tx, entry.GetItem().(*repositories.Repository))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyRepositoryAccessChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.repositoryAccess.ExecuteInsert(tx, entry.GetItem().(*repositories.RepositoryAccess))

	case change.Updated:
		return c.repositoryAccess.ExecuteUpdate(tx, entry.GetItem().(*repositories.RepositoryAccess))

	case change.Deleted:
		return c.repositoryAccess.ExecuteDelete(tx, entry.GetItem().(*repositories.RepositoryAccess))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyManifestChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.manifest.ExecuteInsert(tx, entry.GetItem().(*repositories.Manifest))

	case change.Updated:
		return c.manifest.ExecuteUpdate(tx, entry.GetItem().(*repositories.Manifest))

	case change.Deleted:
		return c.manifest.ExecuteDelete(tx, entry.GetItem().(*repositories.Manifest))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyTagChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.tags.ExecuteInsert(tx, entry.GetItem().(*repositories.Tag))

	case change.Deleted:
		return c.tags.ExecuteDelete(tx, entry.GetItem().(*repositories.Tag))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyBlobChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.blobs.ExecuteInsert(tx, entry.GetItem().(*repositories.Blob))

	case change.Deleted:
		return c.blobs.ExecuteDelete(tx, entry.GetItem().(*repositories.Blob))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyRepositoryBlobChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.repositoryBlobs.ExecuteInsert(tx, entry.GetItem().(*repositories.RepositoryBlob))

	case change.Deleted:
		return c.repositoryBlobs.ExecuteDelete(tx, entry.GetItem().(*repositories.RepositoryBlob))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}

func (c *Context) applyFileChange(tx *memdb.Txn, entry *change.Entry) error {
	switch entry.GetChangeType() {
	case change.Added:
		return c.files.ExecuteInsert(tx, entry.GetItem().(*repositories.File))

	case change.Deleted:
		return c.files.ExecuteDelete(tx, entry.GetItem().(*repositories.File))

	default:
		return fmt.Errorf("unsupported change type: %v", entry.GetChangeType())
	}
}
