package blobStorage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/jsontypes"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/services/kv"
)

// TODO: cleanup temporary blobs of expired sessions

type memoryService struct {
	blobs map[string][]byte
	mu    *sync.RWMutex
}

func NewInMemoryService() Service {
	return &memoryService{
		blobs: make(map[string][]byte),
		mu:    &sync.RWMutex{},
	}
}

func (m *memoryService) StartUploadSession(ctx context.Context, params StartUploadSessionParams) (*StartUploadSessionResponse, error) {
	scope := middlewares.GetScope(ctx)

	digestState, err := sha256.New().(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal digest state: %w", err)
	}

	session := jsontypes.UploadSession{
		Id:             uuid.New(),
		UploadMode:     params.BlobUploadMode,
		TenantSlug:     params.TenantSlug,
		ProjectSlug:    params.ProjectSlug,
		RepositorySlug: params.RepositorySlug,
		RepositoryId:   params.RepositoryId,
		DigestState:    digestState,
		RangeEnd:       0,
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

	m.setTempBlob(session.Id, []byte{})

	return &StartUploadSessionResponse{
		SessionId: session.Id,
	}, nil
}

func (m *memoryService) getTempBlob(sessionId uuid.UUID) []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.blobs["temp:"+sessionId.String()]
}

func (m *memoryService) setTempBlob(sessionId uuid.UUID, blob []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blobs["temp:"+sessionId.String()] = blob
}

func (m *memoryService) removeTempBlob(sessionId uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.blobs, fmt.Sprintf("temp:%s", sessionId))
}

func (m *memoryService) setBlob(digest string, blob []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blobs["blob:"+digest] = blob
}

func (m *memoryService) UploadWriteChunk(ctx context.Context, sessionId uuid.UUID, reader io.Reader, length int64) error {
	scope := middlewares.GetScope(ctx)
	kvStore := ioc.GetDependency[kv.Store](scope)

	value, ok, err := kvStore.Get(ctx, buildSessionCacheKey(sessionId))
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if !ok {
		return fmt.Errorf("session not found")
	}

	var session jsontypes.UploadSession
	err = json.Unmarshal([]byte(value), &session)
	if err != nil {
		return fmt.Errorf("failed to unmarshal session: %w", err)
	}

	blobBytes := m.getTempBlob(sessionId)
	writer := bytes.NewBuffer(blobBytes)

	hasher := sha256.New()
	err = hasher.(encoding.BinaryUnmarshaler).UnmarshalBinary(session.DigestState)
	if err != nil {
		return fmt.Errorf("failed to unmarshal digest state: %w", err)
	}

	multiWriter := io.MultiWriter(writer, hasher)
	_, err = io.CopyN(multiWriter, reader, length)
	if err != nil {
		return fmt.Errorf("failed to copy reader: %w", err)
	}

	digestState, err := hasher.(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal digest state: %w", err)
	}

	session.DigestState = digestState
	session.RangeEnd += length

	jsonBytes, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	err = kvStore.Set(ctx, buildSessionCacheKey(session.Id), string(jsonBytes), kv.WithExpiration(time.Minute*5))
	if err != nil {
		return fmt.Errorf("failed to set session: %w", err)
	}

	m.setTempBlob(sessionId, writer.Bytes())
	return nil
}

func (m *memoryService) CompleteUpload(ctx context.Context, sessionId uuid.UUID) (*CompleteUploadResponse, error) {
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
	m.setBlob(digest, m.getTempBlob(sessionId))
	m.removeTempBlob(sessionId)

	return &CompleteUploadResponse{
		ComputedDigest: digest,
		Size:           session.RangeEnd,
		TenantSlug:     session.TenantSlug,
		ProjectSlug:    session.ProjectSlug,
		RepositorySlug: session.RepositorySlug,
		RepositoryId:   session.RepositoryId,
	}, nil
}

func (m *memoryService) GetUploadRangeEnd(ctx context.Context, sessionId uuid.UUID) (int64, error) {
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
