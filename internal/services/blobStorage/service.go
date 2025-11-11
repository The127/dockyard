package blobStorage

import (
	"context"
	"fmt"
	"io"

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
	TenantSlug     string
	ProjectSlug    string
	RepositorySlug string
	RepositoryId   uuid.UUID
}

type Service interface {
	StartUploadSession(ctx context.Context, params StartUploadSessionParams) (*StartUploadSessionResponse, error)
	UploadWriteChunk(ctx context.Context, sessionId uuid.UUID, reader io.Reader, length int64) error
	CompleteUpload(ctx context.Context, sessionId uuid.UUID) (*CompleteUploadResponse, error)
	GetUploadRangeEnd(ctx context.Context, sessionId uuid.UUID) (int, error)
}

func buildSessionCacheKey(sessionId uuid.UUID) string {
	return fmt.Sprintf("blob_upload_session:%s", sessionId)
}
