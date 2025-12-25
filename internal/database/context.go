package database

import (
	"context"

	"github.com/the127/dockyard/internal/repositories"
)

const (
	TenantType int = iota
	ProjectType
	ProjectAccessType
	UserType
	PatType
	RepositoryType
	RepositoryAccessType
	ManifestType
	TagType
	BlobType
	RepositoryBlobType
	FileType
)

type Context interface {
	Tenants() repositories.TenantRepository
	Projects() repositories.ProjectRepository
	ProjectAccess() repositories.ProjectAccessRepository
	Users() repositories.UserRepository
	Pats() repositories.PatRepository
	Repositories() repositories.RepositoryRepository
	RepositoryAccess() repositories.RepositoryAccessRepository
	Manifests() repositories.ManifestRepository
	Tags() repositories.TagRepository
	Blobs() repositories.BlobRepository
	RepositoryBlobs() repositories.RepositoryBlobRepository
	Files() repositories.FileRepository

	SaveChanges(ctx context.Context)
}
