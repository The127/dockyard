package storageBackends

import (
	"context"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// StorageBackendState represents a map-based structure for tracking the state of a storage backend during operations.
type StorageBackendState map[string]string

// StorageBackend defines the interface for backend storage operations with support for blob uploads, downloads, and deletion.
type StorageBackend interface {

	// InitiateUpload begins a new upload session for a blob, returning the storage backend state and any encountered error.
	InitiateUpload(ctx context.Context, id uuid.UUID, contentType string) (StorageBackendState, error)

	// UploadAddChunk appends a chunk of data from the provided reader to an ongoing upload, updating the state accordingly.
	UploadAddChunk(ctx context.Context, state StorageBackendState, reader io.Reader) (StorageBackendState, error)

	// CompleteUpload finalizes the blob upload by verifying its digest and committing it to storage. Returns an error on failure.
	CompleteUpload(ctx context.Context, digest string, state StorageBackendState) error

	// AbortUpload aborts an ongoing upload and cleans up related state for the specified storage backend operation.
	AbortUpload(ctx context.Context, state StorageBackendState) error

	// DeleteBlob removes a blob identified by the specified digest from the storage backend. Returns an error if deletion fails.
	DeleteBlob(ctx context.Context, digest string) error

	// DownloadBlob retrieves a blob by its digest and writes it to the provided HTTP response writer.
	// It sets the appropriate Content-Type header using the blob's metadata. Returns an error if the operation fails.
	DownloadBlob(ctx context.Context, w http.ResponseWriter, digest string) error
}
