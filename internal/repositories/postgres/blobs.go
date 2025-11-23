package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type blobRepository struct {
}

func NewPostgresBlobRepository() repositories.BlobRepository {
	return &blobRepository{}
}

func (r *blobRepository) First(_ context.Context, _ *repositories.BlobFilter) (*repositories.Blob, error) {
	return nil, nil
}

func (r *blobRepository) Single(_ context.Context, filter *repositories.BlobFilter) (*repositories.Blob, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiBlobNotFound
	}
	return result, nil
}

func (r *blobRepository) List(_ context.Context, _ *repositories.BlobFilter) ([]*repositories.Blob, int, error) {
	return nil, 0, nil
}

func (r *blobRepository) Insert(_ context.Context, _ *repositories.Blob) error {
	return nil
}

func (r *blobRepository) Update(_ context.Context, _ *repositories.Blob) error {
	return nil
}

func (r *blobRepository) Delete(_ context.Context, id uuid.UUID) error {
	return nil
}
