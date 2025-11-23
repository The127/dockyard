package inmemory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type blobRepository struct {
	txn *memdb.Txn
}

func NewInMemoryBlobRepository(txn *memdb.Txn) repositories.BlobRepository {
	return &blobRepository{
		txn: txn,
	}
}

func (r *blobRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.BlobFilter) ([]*repositories.Blob, int, error) {
	var result []*repositories.Blob

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.Blob)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count, nil
}

func (r *blobRepository) matches(blob *repositories.Blob, filter *repositories.BlobFilter) bool {
	if filter.HasId() {
		if blob.GetId() != filter.GetId() {
			return false
		}
	}

	if filter.HasDigest() {
		if blob.GetDigest() != filter.GetDigest() {
			return false
		}
	}

	return true
}

func (r *blobRepository) First(_ context.Context, filter *repositories.BlobFilter) (*repositories.Blob, error) {
	iterator, err := r.txn.Get("blobs", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get blobs: %w", err)
	}

	result, _, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to apply filter: %w", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
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

func (r *blobRepository) List(_ context.Context, filter *repositories.BlobFilter) ([]*repositories.Blob, int, error) {
	iterator, err := r.txn.Get("blobs", "id")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get blobs: %w", err)
	}

	result, count, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to apply filter: %w", err)
	}

	return result, count, nil
}

func (r *blobRepository) Insert(_ context.Context, blob *repositories.Blob) error {
	err := r.txn.Insert("blobs", *blob)
	if err != nil {
		return fmt.Errorf("failed to insert blob: %w", err)
	}

	return nil
}

func (r *blobRepository) Delete(_ context.Context, id uuid.UUID) error {
	entry, err := r.First(context.Background(), repositories.NewBlobFilter().ById(id))
	if err != nil {
		return fmt.Errorf("failed to get by id: %w", err)
	}
	if entry == nil {
		return nil
	}

	err = r.txn.Delete("blobs", entry)
	if err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	return nil
}
