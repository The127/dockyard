package inmemory

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/storageBackends"
	"github.com/the127/dockyard/internal/utils/ociError"
)

type blobInfo struct {
	contentType string
	data        []byte
}

type backend struct {
	blobs   map[string]blobInfo
	temp    map[string]tempState
	blobsMu *sync.RWMutex
	tempMu  *sync.RWMutex
}

type tempState struct {
	buffer *bytes.Buffer
}

func New() storageBackends.StorageBackend {
	return &backend{
		blobs:   make(map[string]blobInfo),
		temp:    make(map[string]tempState),
		blobsMu: &sync.RWMutex{},
		tempMu:  &sync.RWMutex{},
	}
}

func (b *backend) InitiateUpload(ctx context.Context, id uuid.UUID, contentType string) (storageBackends.StorageBackendState, error) {
	b.setTemp(id.String(), &tempState{
		buffer: &bytes.Buffer{},
	})

	return map[string]string{
		"id":           id.String(),
		"content-type": contentType,
	}, nil
}

func (b *backend) UploadAddChunk(ctx context.Context, state storageBackends.StorageBackendState, reader io.Reader) (storageBackends.StorageBackendState, error) {
	id, ok := state["id"]
	if !ok {
		return state, fmt.Errorf("missing id in state")
	}

	tempState := b.getTemp(id)

	if tempState == nil {
		return state, fmt.Errorf("upload not initiated")
	}

	_, err := io.Copy(tempState.buffer, reader)
	return state, err
}

func (b *backend) CompleteUpload(ctx context.Context, digest string, state storageBackends.StorageBackendState) error {
	id, ok := state["id"]
	if !ok {
		return fmt.Errorf("missing id in state")
	}

	contentType, ok := state["content-type"]
	if !ok {
		return fmt.Errorf("missing content-type in state")
	}

	tempState := b.getTemp(id)

	if tempState == nil {
		return fmt.Errorf("upload not initiated")
	}

	b.setBlob(digest, &blobInfo{
		contentType: contentType,
		data:        tempState.buffer.Bytes(),
	})

	return nil
}

func (b *backend) AbortUpload(ctx context.Context, state storageBackends.StorageBackendState) error {
	id, ok := state["id"]
	if !ok {
		return fmt.Errorf("missing id in state")
	}

	b.setTemp(id, nil)

	return nil
}

func (b *backend) DeleteBlob(ctx context.Context, digest string) error {
	b.setBlob(digest, nil)
	return nil
}

func (b *backend) DownloadBlob(ctx context.Context, w http.ResponseWriter, digest string) error {
	blob := b.getBlob(digest)
	if blob == nil {
		return ociError.NewOciError(ociError.BlobUnknown)
	}

	w.Header().Set("Content-Type", blob.contentType)

	_, err := w.Write(blob.data)
	if err != nil {
		return fmt.Errorf("failed to write blob: %w", err)
	}

	return nil
}

func (b *backend) setBlob(digest string, blob *blobInfo) {
	b.blobsMu.Lock()
	defer b.blobsMu.Unlock()

	if blob == nil {
		delete(b.blobs, digest)
	} else {
		b.blobs[digest] = *blob
	}
}

func (b *backend) getBlob(digest string) *blobInfo {
	b.blobsMu.RLock()
	defer b.blobsMu.RUnlock()

	data, ok := b.blobs[digest]
	if !ok {
		return nil
	}
	return &data
}

func (b *backend) setTemp(digest string, state *tempState) {
	b.tempMu.Lock()
	defer b.tempMu.Unlock()

	if state == nil {
		delete(b.temp, digest)
	} else {
		b.temp[digest] = *state
	}
}

func (b *backend) getTemp(digest string) *tempState {
	b.tempMu.RLock()
	defer b.tempMu.RUnlock()

	data, ok := b.temp[digest]
	if !ok {
		return nil
	}
	return &data
}
