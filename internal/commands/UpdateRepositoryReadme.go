package commands

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type UpdateRepositoryReadme struct {
	TenantSlug     string
	ProjectSlug    string
	RepositorySlug string
	Content        []byte
}

type UpdateRepositoryReadmeResponse struct{}

func HandleUpdateRepositoryReadme(ctx context.Context, command UpdateRepositoryReadme) (*UpdateRepositoryReadmeResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[services.DbService](scope)

	tx, err := dbService.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenant, err := tx.Tenants().Single(ctx, repositories.NewTenantFilter().BySlug(command.TenantSlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	project, err := tx.Projects().Single(ctx, repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(command.ProjectSlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	repository, err := tx.Repositories().Single(ctx, repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(command.RepositorySlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	digestBytes := sha256.Sum256(command.Content)
	digest := fmt.Sprintf("sha256:%x", digestBytes[:])

	if repository.GetReadmeFileId() != nil {
		err = tx.Files().Delete(ctx, *repository.GetReadmeFileId())
		if err != nil {
			return nil, fmt.Errorf("failed to delete readme file: %w", err)
		}
	}

	newReadme := repositories.NewFile(digest, "text/markdown", command.Content)
	err = tx.Files().Insert(ctx, newReadme)
	if err != nil {
		return nil, fmt.Errorf("failed to insert readme file: %w", err)
	}

	repository.SetReadmeFileId(pointer.To(newReadme.GetId()))
	err = tx.Repositories().Update(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to update repository: %w", err)
	}

	return &UpdateRepositoryReadmeResponse{}, nil
}
