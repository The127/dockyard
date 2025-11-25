package directory

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/storageBackends"
	"github.com/the127/dockyard/internal/utils"
)

type backend struct {
	path     string
	tempPath string
}

type tempState struct {
	id          string
	filePath    string
	contentType string
}

func (t tempState) encode() storageBackends.StorageBackendState {
	return storageBackends.StorageBackendState{
		"id":          t.id,
		"filePath":    t.filePath,
		"contentType": t.contentType,
	}
}

func decodeState(state storageBackends.StorageBackendState) (tempState, error) {
	id, ok := state["id"]
	if !ok {
		return tempState{}, fmt.Errorf("missing id in state")
	}

	filePath, ok := state["filePath"]
	if !ok {
		return tempState{}, fmt.Errorf("missing filePath in state")
	}

	contentType, ok := state["contentType"]
	if !ok {
		return tempState{}, fmt.Errorf("missing contentType in state")
	}

	return tempState{
		id:          id,
		filePath:    filePath,
		contentType: contentType,
	}, nil
}

func New(c config.DirectoryBlobStorageConfig) (storageBackends.StorageBackend, error) {
	err := os.MkdirAll(c.Path, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("ensuring path exists: %w", err)
	}

	err = os.MkdirAll(c.TempPath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("ensuring temp path exists: %w", err)
	}

	return &backend{
		path:     c.Path,
		tempPath: c.TempPath,
	}, nil
}

func (b *backend) InitiateUpload(_ context.Context, id uuid.UUID, contentType string) (storageBackends.StorageBackendState, error) {
	// create the data file
	filePath := path.Join(b.path, id.String())
	err := os.WriteFile(filePath, []byte{}, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("creating data file: %w", err)
	}

	return tempState{
		id:          id.String(),
		filePath:    filePath,
		contentType: contentType,
	}.encode(), nil
}

func (b *backend) UploadAddChunk(_ context.Context, state storageBackends.StorageBackendState, reader io.Reader) (storageBackends.StorageBackendState, error) {
	decodedState, err := decodeState(state)
	if err != nil {
		return nil, fmt.Errorf("decoding state: %w", err)
	}

	dataFile, err := os.OpenFile(decodedState.filePath, os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("opening data file: %w", err)
	}

	defer utils.PanicOnError(dataFile.Close, "closing data file")

	_, err = io.Copy(dataFile, reader)
	if err != nil {
		return nil, fmt.Errorf("writing chunk to data file: %w", err)
	}

	return decodedState.encode(), nil
}

func (b *backend) CompleteUpload(_ context.Context, digest string, state storageBackends.StorageBackendState) error {
	decodedState, err := decodeState(state)
	if err != nil {
		return fmt.Errorf("decoding state: %w", err)
	}

	dataPath := path.Join(b.path, digest)
	err = os.Rename(decodedState.filePath, dataPath)
	if err != nil {
		return fmt.Errorf("renaming data file: %w", err)
	}

	err = os.WriteFile(dataPath+".info", []byte(decodedState.contentType), os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing info file: %w", err)
	}

	return nil
}

func (b *backend) AbortUpload(_ context.Context, state storageBackends.StorageBackendState) error {
	decodedState, err := decodeState(state)
	if err != nil {
		return fmt.Errorf("decoding state: %w", err)
	}

	err = os.Remove(decodedState.filePath)
	if err != nil {
		return fmt.Errorf("removing data file: %w", err)
	}

	return nil
}

func (b *backend) DeleteBlob(_ context.Context, digest string) error {
	dataPath := path.Join(b.path, digest)
	err := os.Remove(dataPath)
	if err != nil {
		return fmt.Errorf("removing data file: %w", err)
	}

	err = os.Remove(dataPath + ".info")
	if err != nil {
		return fmt.Errorf("removing info file: %w", err)
	}

	return nil
}

func (b *backend) DownloadBlob(_ context.Context, w http.ResponseWriter, digest string) error {
	dataPath := path.Join(b.path, digest)
	infoPath := dataPath + ".info"
	contentType, err := os.ReadFile(infoPath)
	switch {
	case os.IsNotExist(err):
		w.WriteHeader(http.StatusNotFound)
		return nil

	case err != nil:
		return fmt.Errorf("reading info file: %w", err)
	}

	w.Header().Set("Content-Type", string(contentType))

	dataFile, err := os.Open(dataPath)
	if err != nil {
		return fmt.Errorf("opening data file: %w", err)
	}

	defer utils.PanicOnError(dataFile.Close, "closing data file")

	_, err = io.Copy(w, dataFile)
	if err != nil {
		return fmt.Errorf("writing data to response writer: %w", err)
	}

	return nil
}
