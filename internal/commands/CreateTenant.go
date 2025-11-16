package commands

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
)

type CreateTenant struct {
	Slug        string
	DisplayName string

	OidcClient      string
	OidcIssuer      string
	OidcRoleClaim   string
	OidcRoleFormat  string
	OidcRoleMapping map[string]string
}

type CreateTenantResponse struct {
	Id uuid.UUID
}

func HandleCreateTenant(ctx context.Context, command CreateTenant) (*CreateTenantResponse, error) {
	scope := middlewares.GetScope(ctx)

	db := ioc.GetDependency[services.DbService](scope)
	tx, err := db.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenant := repositories.NewTenant(
		command.Slug,
		command.DisplayName,
		repositories.NewTenantOidcConfig(
			command.OidcClient,
			command.OidcIssuer,
			command.OidcRoleClaim,
			command.OidcRoleFormat,
			command.OidcRoleMapping,
		),
	)
	err = tx.Tenants().Insert(ctx, tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to insert tenant: %w", err)
	}

	return &CreateTenantResponse{
		Id: tenant.GetId(),
	}, nil
}
