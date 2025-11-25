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
	temp    map[string]tempBlob
	blobsMu *sync.RWMutex
	tempMu  *sync.RWMutex
}

type tempBlob struct {
	buffer *bytes.Buffer
}

func New() storageBackends.StorageBackend {
	return &backend{
		blobs:   make(map[string]blobInfo),
		temp:    make(map[string]tempBlob),
		blobsMu: &sync.RWMutex{},
		tempMu:  &sync.RWMutex{},
	}
}

type tempState struct {
	id          string
	contentType string
}

func (t tempState) encode() storageBackends.StorageBackendState {
	return storageBackends.StorageBackendState{
		"id":          t.id,
		"contentType": t.contentType,
	}
}

func decodeTempState(state storageBackends.StorageBackendState) (tempState, error) {
	id, ok := state["id"]
	if !ok {
		return tempState{}, fmt.Errorf("missing id in state")
	}

	contentType, ok := state["contentType"]
	if !ok {
		return tempState{}, fmt.Errorf("missing contentType in state")
	}

	return tempState{
		id:          id,
		contentType: contentType,
	}, nil
}

func (b *backend) InitiateUpload(_ context.Context, id uuid.UUID, contentType string) (storageBackends.StorageBackendState, error) {
	b.setTemp(id.String(), &tempBlob{
		buffer: &bytes.Buffer{},
	})

	return tempState{
		id:          id.String(),
		contentType: contentType,
	}.encode(), nil
}

func (b *backend) UploadAddChunk(_ context.Context, state storageBackends.StorageBackendState, reader io.Reader) (storageBackends.StorageBackendState, error) {
	decodedState, err := decodeTempState(state)
	if err != nil {
		return nil, fmt.Errorf("decoding state: %w", err)
	}

	tempData := b.getTemp(decodedState.id)

	if tempData == nil {
		return state, fmt.Errorf("upload not initiated")
	}

	_, err = io.Copy(tempData.buffer, reader)
	if err != nil {
		return nil, fmt.Errorf("copying chunk to buffer: %w", err)
	}

	return state, nil
}

func (b *backend) CompleteUpload(_ context.Context, digest string, state storageBackends.StorageBackendState) error {
	decodedState, err := decodeTempState(state)
	if err != nil {
		return fmt.Errorf("decoding state: %w", err)
	}

	tempData := b.getTemp(decodedState.id)

	if tempData == nil {
		return fmt.Errorf("upload not initiated")
	}

	b.setBlob(digest, &blobInfo{
		contentType: decodedState.contentType,
		data:        tempData.buffer.Bytes(),
	})

	return nil
}

func (b *backend) AbortUpload(_ context.Context, state storageBackends.StorageBackendState) error {
	decodedState, err := decodeTempState(state)
	if err != nil {
		return fmt.Errorf("decoding state: %w", err)
	}

	b.setTemp(decodedState.id, nil)

	return nil
}

func (b *backend) DeleteBlob(_ context.Context, digest string) error {
	b.setBlob(digest, nil)
	return nil
}

func (b *backend) DownloadBlob(_ context.Context, w http.ResponseWriter, digest string) error {
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

func (b *backend) setTemp(digest string, state *tempBlob) {
	b.tempMu.Lock()
	defer b.tempMu.Unlock()

	if state == nil {
		delete(b.temp, digest)
	} else {
		b.temp[digest] = *state
	}
}

func (b *backend) getTemp(digest string) *tempBlob {
	b.tempMu.RLock()
	defer b.tempMu.RUnlock()

	data, ok := b.temp[digest]
	if !ok {
		return nil
	}
	return &data
}
