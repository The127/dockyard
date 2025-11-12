package blobStorage

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/jsontypes"
)

type StartUploadSessionParams struct {
	BlobUploadMode jsontypes.BlobUploadMode
	TenantSlug     string
	ProjectSlug    string
	RepositorySlug string
	RepositoryId   uuid.UUID
}

type StartUploadSessionResponse struct {
	SessionId uuid.UUID
}

type CompleteUploadResponse struct {
	ComputedDigest string
	Size           int64
	TenantSlug     string
	ProjectSlug    string
	RepositorySlug string
	RepositoryId   uuid.UUID
}

type UploadWriteChunkResponse struct {
	Size int64
}

type UploadCompleteBlobResponse struct {
	Size   int64
	Digest string
}

type Service interface {
	StartUploadSession(ctx context.Context, params StartUploadSessionParams) (*StartUploadSessionResponse, error)
	UploadWriteChunk(ctx context.Context, sessionId uuid.UUID, reader io.Reader) (*UploadWriteChunkResponse, error)
	CompleteUpload(ctx context.Context, sessionId uuid.UUID) (*CompleteUploadResponse, error)
	GetUploadRangeEnd(ctx context.Context, sessionId uuid.UUID) (int64, error)

	UploadCompleteBlob(ctx context.Context, reader io.Reader) (*UploadCompleteBlobResponse, error)

	GetBlobDownloadLink(ctx context.Context, digest string) (string, error)
	DownloadBlob(ctx context.Context, w http.ResponseWriter, digest string) error
}

func buildSessionCacheKey(sessionId uuid.UUID) string {
	return fmt.Sprintf("blob_upload_session:%s", sessionId)
}
