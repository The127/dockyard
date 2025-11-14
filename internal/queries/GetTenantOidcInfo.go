package queries

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
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

	return &GetTenantOidcInfoResponse{
		Client: tenant.GetOidcClient(),
		Issuer: tenant.GetOidcIssuer(),
	}, nil
}
