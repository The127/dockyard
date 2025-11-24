package storageBackends

import (
	"context"
	"io"
	"net/http"

	"github.com/google/uuid"
)

type StorageBackendState map[string]string

type StorageBackend interface {
	InitiateUpload(ctx context.Context, id uuid.UUID, contentType string) (StorageBackendState, error)
	UploadAddChunk(ctx context.Context, state StorageBackendState, reader io.Reader) (StorageBackendState, error)
	CompleteUpload(ctx context.Context, digest string, state StorageBackendState) error
	AbortUpload(ctx context.Context, state StorageBackendState) error

	DeleteBlob(ctx context.Context, digest string) error

	DownloadBlob(ctx context.Context, w http.ResponseWriter, digest string) error
}
