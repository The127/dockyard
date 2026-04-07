package queries

import (
	"context"
	"fmt"
	"net/http"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/ociError"
)

type GetRepositoryBlob struct {
	RepositoryId uuid.UUID
	Digest       string
}

type GetRepositoryBlobResponse struct {
	RepositoryBlob *repositories.RepositoryBlob
	Blob           *repositories.Blob
}

func HandleGetRepositoryBlob(ctx context.Context, query GetRepositoryBlob) (*GetRepositoryBlobResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[db.Context](scope)

	blob, err := dbContext.Blobs().First(ctx, repositories.NewBlobFilter().ByDigest(query.Digest))
	if err != nil {
		return nil, fmt.Errorf("getting blob: %w", err)
	}
	if blob == nil {
		return nil, ociError.NewOciError(ociError.BlobUnknown).
			WithMessage(fmt.Sprintf("blob '%s' does not exist", query.Digest)).
			WithHttpCode(http.StatusNotFound)
	}

	repositoryBlob, err := dbContext.RepositoryBlobs().First(ctx, repositories.NewRepositoryBlobFilter().ByBlobId(blob.GetId()).ByRepositoryId(query.RepositoryId))
	if err != nil {
		return nil, fmt.Errorf("getting repository blob: %w", err)
	}
	if repositoryBlob == nil {
		return nil, ociError.NewOciError(ociError.BlobUnknown).
			WithMessage(fmt.Sprintf("blob '%s' does not exist", query.Digest)).
			WithHttpCode(http.StatusNotFound)
	}

	return &GetRepositoryBlobResponse{
		RepositoryBlob: repositoryBlob,
		Blob:           blob,
	}, nil
}
