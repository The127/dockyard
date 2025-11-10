package commands

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/services"
)

type CreateTenant struct {
	Slug        string
	DisplayName string
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

	tx.Tenants()

	return &CreateTenantResponse{
		Id: uuid.New(),
	}, nil
}
