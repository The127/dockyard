package ocihandlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/middlewares/authentication"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func checkAccess(
	ctx context.Context,
	tx database.Transaction,
	repoIdentifier middlewares.OciRepositoryIdentifier,
	repository *repositories.Repository,
	accessType string,
) error {
	var repositoryAccessFilter *repositories.RepositoryAccessFilter
	var repositoryAccess *repositories.RepositoryAccess
	var err error

	if repository.GetIsPublic() {
		return nil
	}

	currentUser := authentication.GetCurrentUser(ctx)
	if currentUser.UserId == uuid.Nil {
		goto failure
	}

	repositoryAccessFilter = repositories.NewRepositoryAccessFilter().
		ByRepositoryId(repository.GetId()).
		ByUserId(currentUser.UserId)
	repositoryAccess, err = tx.RepositoryAccess().First(ctx, repositoryAccessFilter)
	if err != nil {
		return fmt.Errorf("getting repository access: %w", err)
	}
	if repositoryAccess == nil {
		goto failure
	}

	if accessType == "push" {
		if repositoryAccess.GetRole() == repositories.RepositoryAccessRoleGuest {
			goto failure
		}
	}

	// pull access is allowed for all roles

	return nil

failure:
	realm := fmt.Sprintf("%s/v2/token", config.C.Server.ExternalUrl)
	service := fmt.Sprintf("%s:%s", config.C.Server.ExternalDomain, repoIdentifier.TenantSlug)
	scope := fmt.Sprintf("repository:%s/%s/%s:%s", repoIdentifier.TenantSlug, repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, accessType)

	wwwAuthenticateHeaderValue := fmt.Sprintf("Bearer realm=\"%s\",service=\"%s\",scope=\"%s\"", realm, service, scope)

	return ociError.NewOciError(ociError.Unauthorized).
		WithMessage("user is not authenticated").
		WithHttpCode(401).
		WithHeader("WWW-Authenticate", wwwAuthenticateHeaderValue)
}

func getRepositoryByIdentifier(ctx context.Context, tx database.Transaction, repoIdentifier middlewares.OciRepositoryIdentifier) (*repositories.Tenant, *repositories.Project, *repositories.Repository, error) {
	tenant, err := tx.Tenants().First(ctx, repositories.NewTenantFilter().BySlug(repoIdentifier.TenantSlug))
	if err != nil {
		return nil, nil, nil, err
	}
	if tenant == nil {
		return nil, nil, nil, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("tenant '%s' does not exist", repoIdentifier.TenantSlug)).
			WithHttpCode(http.StatusNotFound)
	}

	project, err := tx.Projects().First(ctx, repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(repoIdentifier.ProjectSlug))
	if err != nil {
		return nil, nil, nil, err
	}
	if project == nil {
		return nil, nil, nil, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("project '%s' does not exist", repoIdentifier.ProjectSlug)).
			WithHttpCode(http.StatusNotFound)
	}

	repository, err := tx.Repositories().First(ctx, repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(repoIdentifier.RepositorySlug))
	if err != nil {
		return nil, nil, nil, err
	}
	if repository == nil {
		return nil, nil, nil, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("repository '%s' does not exist", repoIdentifier.RepositorySlug)).
			WithHttpCode(http.StatusNotFound)
	}

	return tenant, project, repository, nil
}
