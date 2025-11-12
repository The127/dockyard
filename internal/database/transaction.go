package database

import "github.com/the127/dockyard/internal/repositories"

type Transaction interface {
	Tenants() repositories.TenantRepository
	Projects() repositories.ProjectRepository
	Users() repositories.UserRepository
	Repositories() repositories.RepositoryRepository
	Manifests() repositories.ManifestRepository
	Tags() repositories.TagRepository
	Blobs() repositories.BlobRepository
	RepositoryBlobs() repositories.RepositoryBlobRepository
	Commit() error
	Rollback() error
}
