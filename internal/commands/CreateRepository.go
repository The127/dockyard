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

type CreateRepository struct {
	UserId      uuid.UUID
	TenantSlug  string
	ProjectSlug string
	Slug        string
	Description *string
	IsPublic    bool
}

type CreateRepositoryResponse struct {
	Id uuid.UUID
}

func HandleCreateRepository(ctx context.Context, command CreateRepository) (*CreateRepositoryResponse, error) {
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

	projectRepository := dbContext.Projects()
	projectFilter := repositories.NewProjectFilter().
		ByTenantId(tenant.GetId()).
		BySlug(command.ProjectSlug)
	project, err := projectRepository.Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	repository := repositories.NewRepository(project.GetId(), command.Slug, fmt.Sprintf("%s/%s", project.GetSlug(), command.Slug))
	repository.SetDescription(command.Description)
	repository.SetIsPublic(command.IsPublic)

	repositoryRepository := dbContext.Repositories()
	repositoryRepository.Insert(repository)

	repositoryAccess := repositories.NewRepositoryAccess(repository.GetId(), command.UserId, repositories.RepositoryAccessRoleAdmin)
	dbContext.RepositoryAccess().Insert(repositoryAccess)

	return &CreateRepositoryResponse{
		Id: repository.GetId(),
	}, nil
}
