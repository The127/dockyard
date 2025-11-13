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

type GetRepository struct {
	TenantSlug     string
	ProjectSlug    string
	RepositorySlug string
}

type GetRepositoryResponse struct {
	Id          uuid.UUID
	Slug        string
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func HandleGetRepository(ctx context.Context, query GetRepository) (*GetRepositoryResponse, error) {
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

	repositoryFilter := repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(query.RepositorySlug)
	repository, err := tx.Repositories().Single(ctx, repositoryFilter)
	if err != nil {
		return nil, fmt.Errorf("getting repository: %w", err)
	}

	return &GetRepositoryResponse{
		Id:          repository.GetId(),
		Slug:        repository.GetSlug(),
		DisplayName: repository.GetDisplayName(),
		CreatedAt:   repository.GetCreatedAt(),
		UpdatedAt:   repository.GetUpdatedAt(),
	}, nil
}
