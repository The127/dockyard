package queries

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
)

type ListTags struct {
	TenantSlug     string
	ProjectSlug    string
	RepositorySlug string
}

type ListTagsResponse PagedResponse[ListTagsResponseItem]

type ListTagsResponseItem struct {
	Id     uuid.UUID
	Name   string
	Digest string
	Size   int64
}

func HandleListTags(ctx context.Context, query ListTags) (*ListTagsResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[db.Context](scope)

	tenant, err := dbContext.Tenants().Single(ctx, repositories.NewTenantFilter().BySlug(query.TenantSlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	project, err := dbContext.Projects().Single(ctx, repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(query.ProjectSlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	repository, err := dbContext.Repositories().Single(ctx, repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(query.RepositorySlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	tags, _, err := dbContext.Tags().List(ctx, repositories.NewTagFilter().ByRepositoryId(repository.GetId()).WithManifestInfo())
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	items := make([]ListTagsResponseItem, len(tags))
	for i, tag := range tags {
		items[i] = ListTagsResponseItem{
			Id:     tag.GetId(),
			Name:   tag.GetName(),
			Digest: tag.GetManifestInfo().Digest,
			Size:   -1,
		}
	}

	return &ListTagsResponse{
		Items: items,
	}, nil
}
