package queries

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
)

type ListTenants struct{}

type ListTenantsResponse PagedResponse[ListTenantsResponseItem]

type ListTenantsResponseItem struct {
	Slug        string
	DisplayName string
}

func HandleListTenants(ctx context.Context, query ListTenants) (*ListTenantsResponse, error) {
	scope := middlewares.GetScope(ctx)

	dbFactory := ioc.GetDependency[db.Factory](scope)
	dbContext, err := dbFactory.NewDbContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenantFilter := repositories.NewTenantFilter()
	tenants, _, err := dbContext.Tenants().List(ctx, tenantFilter)
	if err != nil {
		return nil, fmt.Errorf("listing tenants: %w", err)
	}

	items := make([]ListTenantsResponseItem, len(tenants))
	for i, tenant := range tenants {
		items[i] = ListTenantsResponseItem{
			Slug:        tenant.GetSlug(),
			DisplayName: tenant.GetDisplayName(),
		}
	}

	return &ListTenantsResponse{
		Items: items,
	}, nil
}
