package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type RepositoryBlobRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryRepositoryBlobRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) repositories.RepositoryBlobRepository {
	return &RepositoryBlobRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *RepositoryBlobRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.RepositoryBlobFilter) ([]*repositories.RepositoryBlob, int) {
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

	return result, count
}

func (r *RepositoryBlobRepository) matches(repositoryBlob *repositories.RepositoryBlob, filter *repositories.RepositoryBlobFilter) bool {
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

func (r *RepositoryBlobRepository) First(_ context.Context, filter *repositories.RepositoryBlobFilter) (*repositories.RepositoryBlob, error) {
	iterator, err := r.txn.Get("repository_blobs", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get repository blobs: %w", err)
	}

	result, _ := r.applyFilter(iterator, filter)

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *RepositoryBlobRepository) Single(_ context.Context, filter *repositories.RepositoryBlobFilter) (*repositories.RepositoryBlob, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiRepositoryBlobNotFound
	}
	return result, nil
}

func (r *RepositoryBlobRepository) List(_ context.Context, filter *repositories.RepositoryBlobFilter) ([]*repositories.RepositoryBlob, int, error) {
	iterator, err := r.txn.Get("repository_blobs", "id")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get repository blobs: %w", err)
	}

	result, count := r.applyFilter(iterator, filter)

	return result, count, nil
}

func (r *RepositoryBlobRepository) Insert(_ context.Context, repositoryBlob *repositories.RepositoryBlob) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, repositoryBlob))
	return nil
}

func (r *RepositoryBlobRepository) ExecuteInsert(_ context.Context, repositoryBlob *repositories.RepositoryBlob) error {
	err := r.txn.Insert("repository_blobs", *repositoryBlob)
	if err != nil {
		return fmt.Errorf("failed to insert repository blob: %w", err)
	}

	return nil
}

func (r *RepositoryBlobRepository) Delete(_ context.Context, repositoryBlob *repositories.RepositoryBlob) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, repositoryBlob))
	return nil
}

func (r *RepositoryBlobRepository) ExecuteDelete(_ context.Context, repositoryBlob *repositories.RepositoryBlob) error {
	err := r.txn.Delete("repository_blobs", repositoryBlob)
	if err != nil {
		return fmt.Errorf("failed to delete repository blob: %w", err)
	}

	return nil
}
