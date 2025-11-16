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

	switch repository.GetReadmeFileId() {
	case nil:
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

	default:
		existingReadme, err := tx.Files().Single(ctx, repositories.NewFileFilter().ById(*repository.GetReadmeFileId()))
		if err != nil {
			return nil, fmt.Errorf("failed to get readme file: %w", err)
		}

		existingReadme.Digest = digest
		existingReadme.Data = command.Content
		err = tx.Files().Update(ctx, existingReadme)
		if err != nil {
			return nil, fmt.Errorf("failed to update readme file: %w", err)
		}
	}

	return &UpdateRepositoryReadmeResponse{}, nil
}
