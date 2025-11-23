package queries

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type GetRepositoryReadme struct {
	TenantSlug     string
	ProjectSlug    string
	RepositorySlug string
}

type GetRepositoryReadmeResponse struct {
	Content *[]byte
}

func HandleGetRepositoryReadme(ctx context.Context, query GetRepositoryReadme) (*GetRepositoryReadmeResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[services.DbService](scope)

	tx, err := dbService.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenant, err := tx.Tenants().Single(ctx, repositories.NewTenantFilter().BySlug(query.TenantSlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	project, err := tx.Projects().Single(ctx, repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(query.ProjectSlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	repository, err := tx.Repositories().Single(ctx, repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(query.RepositorySlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	var content *[]byte
	if repository.GetReadmeFileId() != nil {
		file, err := tx.Files().Single(ctx, repositories.NewFileFilter().ById(*repository.GetReadmeFileId()))
		if err != nil {
			return nil, fmt.Errorf("failed to get readme file: %w", err)
		}

		content = pointer.To(file.GetData())
	}

	return &GetRepositoryReadmeResponse{
		Content: content,
	}, nil
}
