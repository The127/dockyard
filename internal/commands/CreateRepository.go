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

type CreateRepository struct {
	TenantSlug  string
	ProjectSlug string
	Slug        string
	Description *string
}

type CreateRepositoryResponse struct {
	Id uuid.UUID
}

func HandleCreateRepository(ctx context.Context, command CreateRepository) (*CreateRepositoryResponse, error) {
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

	projectRepository := tx.Projects()
	projectFilter := repositories.NewProjectFilter().
		ByTenantId(tenant.GetId()).
		BySlug(command.ProjectSlug)
	project, err := projectRepository.Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	repository := repositories.NewRepository(project.GetId(), command.Slug, fmt.Sprintf("%s/%s", project.GetSlug(), command.Slug))
	repository.SetDescription(command.Description)

	repositoryRepository := tx.Repositories()
	err = repositoryRepository.Insert(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to insert repository: %w", err)
	}

	return &CreateRepositoryResponse{
		Id: repository.GetId(),
	}, nil
}
