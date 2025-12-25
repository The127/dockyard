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

type ListRepositories struct {
	TenantSlug  string
	ProjectSlug string
}

type ListRepositoriesResponse PagedResponse[ListRepositoriesResponseItem]

type ListRepositoriesResponseItem struct {
	Id          uuid.UUID
	Slug        string
	DisplayName string
	Description *string
	IsPublic    bool
}

func HandleListRepositories(ctx context.Context, query ListRepositories) (*ListRepositoriesResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[db.Context](scope)

	tenantFilter := repositories.NewTenantFilter().BySlug(query.TenantSlug)
	tenant, err := dbContext.Tenants().Single(ctx, tenantFilter)
	if err != nil {
		return nil, fmt.Errorf("getting tenant: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(query.ProjectSlug)
	project, err := dbContext.Projects().Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	repositoryFilter := repositories.NewRepositoryFilter().ByProjectId(project.GetId())
	repos, _, err := dbContext.Repositories().List(ctx, repositoryFilter)
	if err != nil {
		return nil, fmt.Errorf("listing repositories: %w", err)
	}

	items := make([]ListRepositoriesResponseItem, len(repos))
	for i, repository := range repos {
		items[i] = ListRepositoriesResponseItem{
			Id:          repository.GetId(),
			Slug:        repository.GetSlug(),
			DisplayName: repository.GetDisplayName(),
			Description: repository.GetDescription(),
			IsPublic:    repository.GetIsPublic(),
		}
	}

	return &ListRepositoriesResponse{
		Items: items,
	}, nil
}
