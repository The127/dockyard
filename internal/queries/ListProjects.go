package queries

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
)

type ListProjects struct {
	TenantSlug string
}

type ListProjectsResponse PagedResponse[ListProjectsResponseItem]

type ListProjectsResponseItem struct {
	Slug        string
	DisplayName string
	Description *string
}

func HandleListProjects(ctx context.Context, query ListProjects) (*ListProjectsResponse, error) {
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

	projectFilter := repositories.NewProjectFilter().ByTenantId(tenant.GetId())
	projects, _, err := tx.Projects().List(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	items := make([]ListProjectsResponseItem, len(projects))
	for i, project := range projects {
		items[i] = ListProjectsResponseItem{
			Slug:        project.GetSlug(),
			DisplayName: project.GetDisplayName(),
			Description: project.GetDescription(),
		}
	}

	return &ListProjectsResponse{
		Items: items,
	}, nil
}
