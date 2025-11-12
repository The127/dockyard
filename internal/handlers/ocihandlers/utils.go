package ocihandlers

import (
	"context"
	"fmt"

	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func getRepositoryByIdentifier(ctx context.Context, tx database.Transaction, repoIdentifier middlewares.OciRepositoryIdentifier) (*repositories.Repository, error) {
	tenant, err := tx.Tenants().First(ctx, repositories.NewTenantFilter().BySlug(repoIdentifier.TenantSlug))
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("tenant '%s' does not exist", repoIdentifier.TenantSlug))
	}

	project, err := tx.Projects().First(ctx, repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(repoIdentifier.ProjectSlug))
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("project '%s' does not exist", repoIdentifier.ProjectSlug))
	}

	repository, err := tx.Repositories().First(ctx, repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(repoIdentifier.RepositorySlug))
	if err != nil {
		return nil, err
	}
	if repository == nil {
		return nil, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("repository '%s' does not exist", repoIdentifier.RepositorySlug))
	}

	return repository, nil
}
