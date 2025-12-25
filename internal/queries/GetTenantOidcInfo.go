package queries

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
)

type GetTenantOidcInfo struct {
	TenantSlug string
}

type GetTenantOidcInfoResponse struct {
	Client string
	Issuer string
}

func HandleGetTenantOidcInfo(ctx context.Context, query GetTenantOidcInfo) (*GetTenantOidcInfoResponse, error) {
	scope := middlewares.GetScope(ctx)

	dbFactory := ioc.GetDependency[db.Factory](scope)
	dbContext, err := dbFactory.NewDbContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenantFilter := repositories.NewTenantFilter().BySlug(query.TenantSlug)
	tenant, err := dbContext.Tenants().Single(ctx, tenantFilter)
	if err != nil {
		return nil, fmt.Errorf("getting tenant: %w", err)
	}

	return &GetTenantOidcInfoResponse{
		Client: tenant.GetOidcClient(),
		Issuer: tenant.GetOidcIssuer(),
	}, nil
}
