package queries

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
)

type ListRepositories struct {
	TenantSlug  string
	ProjectSlug string
}

type ListRepositoriesResponse PagedResponse[ListRepositoriesResponseItem]

type ListRepositoriesResponseItem struct {
	Slug        string
	DisplayName string
}

func HandleListRepositories(ctx context.Context, query ListRepositories) (*ListRepositoriesResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[services.DbService](scope)

	tx, err := dbService.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenantFilter := repositories.NewTenantFilter().BySlug(query.TenantSlug)
	tenant, err := tx.Tenants().Single(ctx, tenantFilter)
	if err != nil {
		return nil, fmt.Errorf("getting tenant: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(query.ProjectSlug)
	project, err := tx.Projects().Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	repositoryFilter := repositories.NewRepositoryFilter().ByProjectId(project.GetId())
	repos, _, err := tx.Repositories().List(ctx, repositoryFilter)
	if err != nil {
		return nil, fmt.Errorf("listing repositories: %w", err)
	}

	items := make([]ListRepositoriesResponseItem, len(repos))
	for i, repository := range repos {
		items[i] = ListRepositoriesResponseItem{
			Slug:        repository.GetSlug(),
			DisplayName: repository.GetDisplayName(),
		}
	}

	return &ListRepositoriesResponse{
		Items: items,
	}, nil
}
