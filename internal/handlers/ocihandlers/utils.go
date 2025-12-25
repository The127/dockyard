package ocihandlers

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/middlewares/ociAuthentication"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func checkAccess(
	ctx context.Context,
	repoIdentifier middlewares.OciRepositoryIdentifier,
	accessType ociAuthentication.Access,
) error {
	currentUser := ociAuthentication.GetCurrentUser(ctx)
	if currentUser.Repository != nil {
		if currentUser.Repository.Equals(repoIdentifier) {
			if slices.Contains(currentUser.Access, accessType) {
				return nil
			}
		}
	}

	realm := fmt.Sprintf("%s/v2/token", config.C.Server.ExternalUrl)
	service := fmt.Sprintf("%s:%s", config.C.Server.ExternalDomain, repoIdentifier.TenantSlug)
	scope := fmt.Sprintf("repository:%s/%s/%s:%s", repoIdentifier.TenantSlug, repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, accessType)

	wwwAuthenticateHeaderValue := fmt.Sprintf("Bearer realm=\"%s\",service=\"%s\",scope=\"%s\"", realm, service, scope)

	return ociError.NewOciError(ociError.Unauthorized).
		WithMessage("user is not authenticated").
		WithHttpCode(401).
		WithHeader("WWW-Authenticate", wwwAuthenticateHeaderValue)
}

func getRepositoryByIdentifier(ctx context.Context, dbContext database.Context, repoIdentifier middlewares.OciRepositoryIdentifier) (*repositories.Tenant, *repositories.Project, *repositories.Repository, error) {
	tenant, err := dbContext.Tenants().First(ctx, repositories.NewTenantFilter().BySlug(repoIdentifier.TenantSlug))
	if err != nil {
		return nil, nil, nil, err
	}
	if tenant == nil {
		return nil, nil, nil, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("tenant '%s' does not exist", repoIdentifier.TenantSlug)).
			WithHttpCode(http.StatusNotFound)
	}

	project, err := dbContext.Projects().First(ctx, repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(repoIdentifier.ProjectSlug))
	if err != nil {
		return nil, nil, nil, err
	}
	if project == nil {
		return nil, nil, nil, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("project '%s' does not exist", repoIdentifier.ProjectSlug)).
			WithHttpCode(http.StatusNotFound)
	}

	repository, err := dbContext.Repositories().First(ctx, repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(repoIdentifier.RepositorySlug))
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
