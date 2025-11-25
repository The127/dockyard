package blobStorage

import (
	"context"
	"crypto/sha256"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/jsontypes"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/services/kv"
	"github.com/the127/dockyard/internal/storageBackends"
	"github.com/the127/dockyard/internal/utils/ociError"
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
	Digest string
}

type BlobContentType string

const (
	BlobContentTypeOctetStream BlobContentType = "application/octet-stream"
	BlobContentTypeManifest    BlobContentType = "application/vnd.oci.image.manifest.v1+json"
)

type Service interface {
	StartUploadSession(ctx context.Context, params StartUploadSessionParams) (*StartUploadSessionResponse, error)
	UploadWriteChunk(ctx context.Context, sessionId uuid.UUID, reader io.Reader) (*UploadWriteChunkResponse, error)
	CompleteUpload(ctx context.Context, sessionId uuid.UUID, digest string) (*CompleteUploadResponse, error)
	GetUploadRangeEnd(ctx context.Context, sessionId uuid.UUID) (int64, error)

	UploadCompleteBlob(ctx context.Context, digest string, reader io.Reader, contentType BlobContentType) (*UploadCompleteBlobResponse, error)

	DeleteBlob(ctx context.Context, digest string) error

	GetBlobDownloadLink(ctx context.Context, digest string) (string, error)
	DownloadBlob(ctx context.Context, w http.ResponseWriter, digest string) error
}

func buildSessionCacheKey(sessionId uuid.UUID) string {
	return fmt.Sprintf("blob_upload_session:%s", sessionId)
}

type service struct {
	backend storageBackends.StorageBackend
}

func NewBlobStorageService(backend storageBackends.StorageBackend) Service {
	return &service{
		backend: backend,
	}
}

func (s *service) StartUploadSession(ctx context.Context, params StartUploadSessionParams) (*StartUploadSessionResponse, error) {
	scope := middlewares.GetScope(ctx)

	digestState, err := sha256.New().(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal digest state: %w", err)
	}

	id := uuid.New()

	backendState, err := s.backend.InitiateUpload(ctx, id, string(BlobContentTypeOctetStream))
	if err != nil {
		return nil, err
	}

	session := jsontypes.UploadSession{
		Id:             id,
		UploadMode:     params.BlobUploadMode,
		TenantSlug:     params.TenantSlug,
		ProjectSlug:    params.ProjectSlug,
		RepositorySlug: params.RepositorySlug,
		RepositoryId:   params.RepositoryId,
		DigestState:    digestState,
		RangeEnd:       0,
		BackendState:   backendState,
	}

	jsonBytes, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	kvStore := ioc.GetDependency[kv.Store](scope)
	err = kvStore.Set(ctx, buildSessionCacheKey(session.Id), string(jsonBytes), kv.WithExpiration(time.Minute*5))
	if err != nil {
		return nil, fmt.Errorf("failed to set session: %w", err)
	}

	return &StartUploadSessionResponse{
		SessionId: session.Id,
	}, nil
}

type countReader struct {
	io.Reader
	count int64
}

func (r *countReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.count += int64(n)
	return
}

func (s *service) UploadWriteChunk(ctx context.Context, sessionId uuid.UUID, reader io.Reader) (*UploadWriteChunkResponse, error) {
	scope := middlewares.GetScope(ctx)
	kvStore := ioc.GetDependency[kv.Store](scope)

	value, ok, err := kvStore.Get(ctx, buildSessionCacheKey(sessionId))
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	var session jsontypes.UploadSession
	err = json.Unmarshal([]byte(value), &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	hasher := sha256.New()
	err = hasher.(encoding.BinaryUnmarshaler).UnmarshalBinary(session.DigestState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal digest state: %w", err)
	}

	teeReader := io.TeeReader(reader, hasher)
	countReader := &countReader{teeReader, 0}

	newBackendState, err := s.backend.UploadAddChunk(ctx, session.BackendState, countReader)
	if err != nil {
		return nil, err
	}

	digestState, err := hasher.(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal digest state: %w", err)
	}

	session.DigestState = digestState
	session.BackendState = newBackendState
	session.RangeEnd += countReader.count

	jsonBytes, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	err = kvStore.Set(ctx, buildSessionCacheKey(session.Id), string(jsonBytes), kv.WithExpiration(time.Minute*5))
	if err != nil {
		return nil, fmt.Errorf("failed to set session: %w", err)
	}

	return &UploadWriteChunkResponse{
		Size: session.RangeEnd,
	}, nil
}

func (s *service) CompleteUpload(ctx context.Context, sessionId uuid.UUID, expectedDigest string) (*CompleteUploadResponse, error) {
	scope := middlewares.GetScope(ctx)
	kvStore := ioc.GetDependency[kv.Store](scope)

	value, ok, err := kvStore.Get(ctx, buildSessionCacheKey(sessionId))
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	var session jsontypes.UploadSession
	err = json.Unmarshal([]byte(value), &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	hash := sha256.New()
	err = hash.(encoding.BinaryUnmarshaler).UnmarshalBinary(session.DigestState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal digest state: %w", err)
	}

	digest := fmt.Sprintf("sha256:%x", hash.Sum(nil))

	if expectedDigest != digest {
		err := s.backend.AbortUpload(ctx, session.BackendState)
		if err != nil {
			return nil, fmt.Errorf("failed to abort upload: %w, additional error occurred while aborting: %w", ociError.NewOciError(ociError.DigestInvalid), err)
		}
		return nil, ociError.NewOciError(ociError.DigestInvalid)
	}

	err = s.backend.CompleteUpload(ctx, digest, session.BackendState)
	if err != nil {
		return nil, err
	}

	return &CompleteUploadResponse{
		ComputedDigest: digest,
		Size:           session.RangeEnd,
		TenantSlug:     session.TenantSlug,
		ProjectSlug:    session.ProjectSlug,
		RepositorySlug: session.RepositorySlug,
		RepositoryId:   session.RepositoryId,
	}, nil
}

func (s *service) GetUploadRangeEnd(ctx context.Context, sessionId uuid.UUID) (int64, error) {
	scope := middlewares.GetScope(ctx)
	kvStore := ioc.GetDependency[kv.Store](scope)

	value, ok, err := kvStore.Get(ctx, buildSessionCacheKey(sessionId))
	if err != nil {
		return 0, fmt.Errorf("failed to get session: %w", err)
	}
	if !ok {
		return 0, fmt.Errorf("session not found")
	}

	var session jsontypes.UploadSession
	err = json.Unmarshal([]byte(value), &session)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return session.RangeEnd, nil
}

func (s *service) UploadCompleteBlob(ctx context.Context, digest string, reader io.Reader, contentType BlobContentType) (*UploadCompleteBlobResponse, error) {
	uploadState, err := s.backend.InitiateUpload(ctx, uuid.New(), string(contentType))
	if err != nil {
		return nil, fmt.Errorf("failed to initiate upload: %w", err)
	}

	hash := sha256.New()

	teeReader := io.TeeReader(reader, hash)

	uploadState, err = s.backend.UploadAddChunk(ctx, uploadState, teeReader)
	if err != nil {
		return nil, fmt.Errorf("failed to upload chunk: %w", err)
	}

	gotSha256 := hash.Sum(nil)
	gotDigest := fmt.Sprintf("sha256:%x", gotSha256)

	if gotDigest != digest {
		err := s.backend.AbortUpload(ctx, uploadState)
		if err != nil {
			return nil, fmt.Errorf("failed to abort upload: %w, additional error occurred while aborting: %w", ociError.NewOciError(ociError.DigestInvalid), err)
		}

		return nil, ociError.NewOciError(ociError.DigestInvalid)
	}

	err = s.backend.CompleteUpload(ctx, digest, uploadState)
	if err != nil {
		return nil, fmt.Errorf("failed to complete upload: %w", err)
	}

	return &UploadCompleteBlobResponse{
		Digest: digest,
	}, nil
}

func (s *service) DeleteBlob(ctx context.Context, digest string) error {
	// TODO implement me
	panic("implement me")
}

func (s *service) GetBlobDownloadLink(ctx context.Context, digest string) (string, error) {
	// TODO: signed url
	return fmt.Sprintf("/blobs/api/v1/%s", digest), nil
}

func (s *service) DownloadBlob(ctx context.Context, w http.ResponseWriter, digest string) error {
	// TODO: check url signature?
	err := s.backend.DownloadBlob(ctx, w, digest)
	if err != nil {
		return err
	}

	return nil
}
