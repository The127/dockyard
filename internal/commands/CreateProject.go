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

type CreateProject struct {
	UserId      uuid.UUID
	TenantSlug  string
	Slug        string
	DisplayName string

	Description *string
}

type CreateProjectResponse struct {
	Id uuid.UUID
}

func HandleCreateProject(ctx context.Context, command CreateProject) (*CreateProjectResponse, error) {
	scope := middlewares.GetScope(ctx)

	db := ioc.GetDependency[services.DbService](scope)
	tx, err := db.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenantRepository := tx.Tenants()
	tenantFilter := repositories.NewTenantFilter().
		BySlug(command.TenantSlug)
	tenant, err := tenantRepository.Single(ctx, tenantFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	project := repositories.NewProject(tenant.GetId(), command.Slug, command.DisplayName)
	project.SetDescription(command.Description)

	projectRepository := tx.Projects()
	err = projectRepository.Insert(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to insert project: %w", err)
	}

	projectAccess := repositories.NewProjectAccess(project.GetId(), command.UserId, repositories.ProjectAccessRoleAdmin)
	err = tx.ProjectAccess().Insert(ctx, projectAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to insert project access: %w", err)
	}

	return &CreateProjectResponse{
		Id: project.GetId(),
	}, nil
}
