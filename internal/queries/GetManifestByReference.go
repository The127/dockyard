package queries

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/ociError"
)

type GetManifestByReference struct {
	RepositoryId uuid.UUID
	Reference    string
}

type GetManifestByReferenceResponse struct {
	Manifest *repositories.Manifest
	Blob     *repositories.Blob
}

func HandleGetManifestByReference(ctx context.Context, query GetManifestByReference) (*GetManifestByReferenceResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[db.Context](scope)

	var manifestFilter *repositories.ManifestFilter
	if !strings.HasPrefix(query.Reference, "sha256:") {
		tag, err := dbContext.Tags().First(ctx, repositories.NewTagFilter().ByRepositoryId(query.RepositoryId).ByName(query.Reference))
		if err != nil {
			return nil, fmt.Errorf("getting tag: %w", err)
		}
		if tag == nil {
			return nil, ociError.NewOciError(ociError.ManifestUnknown).
				WithMessage(fmt.Sprintf("tag '%s' does not exist", query.Reference)).
				WithHttpCode(http.StatusNotFound)
		}

		manifestFilter = repositories.NewManifestFilter().ById(tag.GetRepositoryManifestId())
	} else {
		manifestFilter = repositories.NewManifestFilter().ByRepositoryId(query.RepositoryId).ByDigest(query.Reference)
	}

	manifest, err := dbContext.Manifests().First(ctx, manifestFilter)
	if err != nil {
		return nil, fmt.Errorf("getting manifest: %w", err)
	}
	if manifest == nil {
		return nil, ociError.NewOciError(ociError.ManifestUnknown).
			WithMessage(fmt.Sprintf("manifest '%s' does not exist", query.Reference)).
			WithHttpCode(http.StatusNotFound)
	}

	blob, err := dbContext.Blobs().First(ctx, repositories.NewBlobFilter().ById(manifest.GetBlobId()))
	if err != nil {
		return nil, fmt.Errorf("getting blob: %w", err)
	}
	if blob == nil {
		return nil, ociError.NewOciError(ociError.BlobUnknown).
			WithMessage(fmt.Sprintf("blob '%s' does not exist", manifest.GetBlobId()))
	}

	return &GetManifestByReferenceResponse{
		Manifest: manifest,
		Blob:     blob,
	}, nil
}
