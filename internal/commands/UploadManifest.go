package commands

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services/blobStorage"
	"github.com/the127/dockyard/internal/utils/ociError"
)

type UploadManifest struct {
	RepositoryId uuid.UUID
	Reference    string
	Digest       string
	MediaType    string
	Body         []byte
}

type UploadManifestResponse struct {
	Digest string
}

func HandleUploadManifest(ctx context.Context, command UploadManifest) (*UploadManifestResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[db.Context](scope)

	blobService := ioc.GetDependency[blobStorage.Service](scope)
	uploadResponse, err := blobService.UploadCompleteBlob(ctx, command.Digest, bytes.NewReader(command.Body), blobStorage.BlobContentType(command.MediaType))
	if err != nil {
		return nil, fmt.Errorf("uploading blob: %w", err)
	}

	blob, err := getOrCreateBlob(ctx, dbContext, uploadResponse.Digest, int64(len(command.Body)))
	if err != nil {
		return nil, err
	}

	dbContext.RepositoryBlobs().Insert(repositories.NewRepositoryBlob(command.RepositoryId, blob.GetId()))

	if strings.HasPrefix(command.Reference, "sha256:") && command.Reference != command.Digest {
		return nil, ociError.NewOciError(ociError.DigestInvalid)
	}

	manifest, err := dbContext.Manifests().First(ctx, repositories.NewManifestFilter().ByRepositoryId(command.RepositoryId).ByDigest(uploadResponse.Digest))
	if err != nil {
		return nil, fmt.Errorf("getting manifest: %w", err)
	}
	if manifest == nil {
		manifest = repositories.NewManifest(command.RepositoryId, blob.GetId(), uploadResponse.Digest, command.MediaType)
		dbContext.Manifests().Insert(manifest)
	}

	if !strings.HasPrefix(command.Reference, "sha256:") {
		dbContext.Tags().Insert(repositories.NewTag(command.RepositoryId, manifest.GetId(), command.Reference))
	}

	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("saving changes: %w", err)
	}

	return &UploadManifestResponse{
		Digest: uploadResponse.Digest,
	}, nil
}
