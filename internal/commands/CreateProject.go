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
	dbContext := ioc.GetDependency[db.Context](scope)

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
	projectRepository.Insert(project)

	projectAccess := repositories.NewProjectAccess(project.GetId(), command.UserId, repositories.ProjectAccessRoleAdmin)
	dbContext.ProjectAccess().Insert(projectAccess)

	return &CreateProjectResponse{
		Id: project.GetId(),
	}, nil
}
