package commands

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/The127/ioc"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
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
	dbContext := ioc.GetDependency[db.Context](scope)

	tenant, err := dbContext.Tenants().Single(ctx, repositories.NewTenantFilter().BySlug(command.TenantSlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	project, err := dbContext.Projects().Single(ctx, repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(command.ProjectSlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	repository, err := dbContext.Repositories().Single(ctx, repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(command.RepositorySlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	digestBytes := sha256.Sum256(command.Content)
	digest := fmt.Sprintf("sha256:%x", digestBytes[:])

	newReadme := repositories.NewFile(digest, "text/markdown", command.Content)
	dbContext.Files().Insert(newReadme)

	oldReadmeId := repository.GetReadmeFileId()
	repository.SetReadmeFileId(pointer.To(newReadme.GetId()))
	dbContext.Repositories().Update(repository)

	if oldReadmeId != nil {
		oldFile, err := dbContext.Files().First(ctx, repositories.NewFileFilter().ById(*oldReadmeId))
		if err != nil {
			return nil, fmt.Errorf("failed to get readme file: %w", err)
		}

		dbContext.Files().Delete(oldFile)
	}

	return &UpdateRepositoryReadmeResponse{}, nil
}
