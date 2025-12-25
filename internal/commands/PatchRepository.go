package commands

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
)

type PatchRepository struct {
	TenantSlug     string
	ProjectSlug    string
	RepositorySlug string

	Description *string
	IsPublic    *bool
}

type PatchRepositoryResponse struct{}

func HandlePatchRepository(ctx context.Context, command PatchRepository) (*PatchRepositoryResponse, error) {
	scope := middlewares.GetScope(ctx)

	dbService := ioc.GetDependency[db.Factory](scope)
	dbContext, err := dbService.NewDbContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	_, _, repository, err := getRepository(ctx, dbContext, command.TenantSlug, command.ProjectSlug, command.RepositorySlug)
	if err != nil {
		return nil, fmt.Errorf("getting repository: %w", err)
	}

	if command.Description != nil {
		repository.SetDescription(command.Description)
	}

	if command.IsPublic != nil {
		repository.SetIsPublic(*command.IsPublic)
	}

	err = dbContext.Repositories().Update(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to update repository: %w", err)
	}

	return nil, nil
}
