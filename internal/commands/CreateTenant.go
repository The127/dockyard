package commands

import (
	"context"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
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
	dbContext := ioc.GetDependency[db.Context](scope)

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
	dbContext.Tenants().Insert(tenant)

	return &CreateTenantResponse{
		Id: tenant.GetId(),
	}, nil
}
