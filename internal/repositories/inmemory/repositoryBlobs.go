package inmemory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type repositoryBlobRepository struct {
	txn *memdb.Txn
}

func NewInMemoryRepositoryBlobRepository(txn *memdb.Txn) repositories.RepositoryBlobRepository {
	return &repositoryBlobRepository{
		txn: txn,
	}
}

func (r *repositoryBlobRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.RepositoryBlobFilter) ([]*repositories.RepositoryBlob, int, error) {
	var result []*repositories.RepositoryBlob

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.RepositoryBlob)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count, nil
}

func (r *repositoryBlobRepository) matches(repositoryBlob *repositories.RepositoryBlob, filter *repositories.RepositoryBlobFilter) bool {
	if filter.HasRepositoryId() {
		if repositoryBlob.GetRepositoryId() != filter.GetRepositoryId() {
			return false
		}
	}

	if filter.HasBlobId() {
		if repositoryBlob.GetBlobId() != filter.GetBlobId() {
			return false
		}
	}

	if filter.HasId() {
		if repositoryBlob.GetId() != filter.GetId() {
			return false
		}
	}

	return true
}

func (r *repositoryBlobRepository) First(_ context.Context, filter *repositories.RepositoryBlobFilter) (*repositories.RepositoryBlob, error) {
	iterator, err := r.txn.Get("repository_blobs", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get repository blobs: %w", err)
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

func (r *repositoryBlobRepository) Single(_ context.Context, filter *repositories.RepositoryBlobFilter) (*repositories.RepositoryBlob, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiRepositoryBlobNotFound
	}
	return result, nil
}

func (r *repositoryBlobRepository) List(_ context.Context, filter *repositories.RepositoryBlobFilter) ([]*repositories.RepositoryBlob, int, error) {
	iterator, err := r.txn.Get("repository_blobs", "id")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get repository blobs: %w", err)
	}

	result, count, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to apply filter: %w", err)
	}

	return result, count, nil
}

func (r *repositoryBlobRepository) Insert(_ context.Context, repositoryBlob *repositories.RepositoryBlob) error {
	err := r.txn.Insert("repository_blobs", *repositoryBlob)
	if err != nil {
		return fmt.Errorf("failed to insert repository blob: %w", err)
	}

	return nil
}

func (r *repositoryBlobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	entry, err := r.First(ctx, repositories.NewRepositoryBlobFilter().ById(id))
	if err != nil {
		return fmt.Errorf("failed to get by id: %w", err)
	}
	if entry == nil {
		return nil
	}

	err = r.txn.Delete("repository_blobs", entry)
	if err != nil {
		return fmt.Errorf("failed to delete repository blob: %w", err)
	}

	return nil
}
