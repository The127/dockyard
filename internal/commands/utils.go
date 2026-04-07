package commands

import (
	"context"
	"fmt"

	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/repositories"
)

func getOrCreateBlob(ctx context.Context, dbContext database.Context, digest string, size int64) (*repositories.Blob, error) {
	blob, err := dbContext.Blobs().First(ctx, repositories.NewBlobFilter().ByDigest(digest))
	if err != nil {
		return nil, fmt.Errorf("getting blob: %w", err)
	}
	if blob == nil {
		blob = repositories.NewBlob(digest, size)
		dbContext.Blobs().Insert(blob)
	}
	return blob, nil
}

func getRepository(ctx context.Context, dbContext database.Context, tenantSlug, projectSlug, repositorySlug string) (*repositories.Tenant, *repositories.Project, *repositories.Repository, error) {
	tenant, err := dbContext.Tenants().Single(ctx, repositories.NewTenantFilter().BySlug(tenantSlug))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	project, err := dbContext.Projects().Single(ctx, repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(projectSlug))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get project: %w", err)
	}

	repository, err := dbContext.Repositories().Single(ctx, repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(repositorySlug))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return tenant, project, repository, nil
}
