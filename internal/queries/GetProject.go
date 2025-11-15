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

type GetProject struct {
	TenantSlug  string
	ProjectSlug string
}

type GetProjectResponse struct {
	Id          uuid.UUID
	Slug        string
	DisplayName string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func HandleGetProject(ctx context.Context, query GetProject) (*GetProjectResponse, error) {
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

	return &GetProjectResponse{
		Id:          project.GetId(),
		Slug:        project.GetSlug(),
		DisplayName: project.GetDisplayName(),
		Description: project.GetDescription(),
		CreatedAt:   project.GetCreatedAt(),
		UpdatedAt:   project.GetUpdatedAt(),
	}, nil
}
