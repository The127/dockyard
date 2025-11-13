package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
)

type GetTenant struct {
	Slug string
}

type GetTenantResponse struct {
	Id          uuid.UUID
	Slug        string
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func HandleGetTenant(ctx context.Context, query GetTenant) (*GetTenantResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[services.DbService](scope)

	tx, err := dbService.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenantFilter := repositories.NewTenantFilter().BySlug(query.Slug)
	tenant, err := tx.Tenants().Single(ctx, tenantFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return &GetTenantResponse{
		Id:          tenant.GetId(),
		Slug:        tenant.GetSlug(),
		DisplayName: tenant.GetDisplayName(),
		CreatedAt:   tenant.GetCreatedAt(),
		UpdatedAt:   tenant.GetUpdatedAt(),
	}, nil
}
