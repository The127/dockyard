package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
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
	Description *string
	IsPublic    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func HandleGetRepository(ctx context.Context, query GetRepository) (*GetRepositoryResponse, error) {
	scope := middlewares.GetScope(ctx)

	dbFactory := ioc.GetDependency[db.Factory](scope)
	dbContext, err := dbFactory.NewDbContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenantFilter := repositories.NewTenantFilter().BySlug(query.TenantSlug)
	tenant, err := dbContext.Tenants().Single(ctx, tenantFilter)
	if err != nil {
		return nil, fmt.Errorf("getting tenant: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(query.ProjectSlug)
	project, err := dbContext.Projects().Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	repositoryFilter := repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(query.RepositorySlug)
	repository, err := dbContext.Repositories().Single(ctx, repositoryFilter)
	if err != nil {
		return nil, fmt.Errorf("getting repository: %w", err)
	}

	return &GetRepositoryResponse{
		Id:          repository.GetId(),
		Slug:        repository.GetSlug(),
		DisplayName: repository.GetDisplayName(),
		Description: repository.GetDescription(),
		IsPublic:    repository.GetIsPublic(),
		CreatedAt:   repository.GetCreatedAt(),
		UpdatedAt:   repository.GetUpdatedAt(),
	}, nil
}
