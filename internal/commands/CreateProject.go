package commands

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
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

	dbFactory := ioc.GetDependency[db.Factory](scope)
	dbContext, err := dbFactory.NewDbContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenantRepository := dbContext.Tenants()
	tenantFilter := repositories.NewTenantFilter().
		BySlug(command.TenantSlug)
	tenant, err := tenantRepository.Single(ctx, tenantFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	project := repositories.NewProject(tenant.GetId(), command.Slug, command.DisplayName)
	project.SetDescription(command.Description)

	projectRepository := dbContext.Projects()
	err = projectRepository.Insert(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to insert project: %w", err)
	}

	projectAccess := repositories.NewProjectAccess(project.GetId(), command.UserId, repositories.ProjectAccessRoleAdmin)
	err = dbContext.ProjectAccess().Insert(ctx, projectAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to insert project access: %w", err)
	}

	return &CreateProjectResponse{
		Id: project.GetId(),
	}, nil
}
