package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type RepositoryRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryRepositoryRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) *RepositoryRepository {
	return &RepositoryRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *RepositoryRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.RepositoryFilter) ([]*repositories.Repository, int) {
	var result []*repositories.Repository

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.Repository)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count
}

func (r *RepositoryRepository) matches(repository *repositories.Repository, filter *repositories.RepositoryFilter) bool {
	if filter.HasSlug() {
		if repository.GetSlug() != filter.GetSlug() {
			return false
		}
	}

	if filter.HasId() {
		if repository.GetId() != filter.GetId() {
			return false
		}
	}

	if filter.HasProjectId() {
		if repository.GetProjectId() != filter.GetProjectId() {
			return false
		}
	}

	return true
}

func (r *RepositoryRepository) First(_ context.Context, filter *repositories.RepositoryFilter) (*repositories.Repository, error) {
	iterator, err := r.txn.Get("repositories", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	result, _ := r.applyFilter(iterator, filter)

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *RepositoryRepository) Single(_ context.Context, filter *repositories.RepositoryFilter) (*repositories.Repository, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiRepositoryNotFound
	}
	return result, nil
}

func (r *RepositoryRepository) List(_ context.Context, filter *repositories.RepositoryFilter) ([]*repositories.Repository, int, error) {
	iterator, err := r.txn.Get("repositories", "id")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get repositories: %w", err)
	}

	result, count := r.applyFilter(iterator, filter)

	return result, count, nil
}

func (r *RepositoryRepository) Insert(_ context.Context, repository *repositories.Repository) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, repository))
	return nil
}

func (r *RepositoryRepository) ExecuteInsert(_ context.Context, repository *repositories.Repository) error {
	err := r.txn.Insert("repositories", *repository)
	if err != nil {
		return fmt.Errorf("failed to insert repository: %w", err)
	}

	repository.ClearChanges()
	return nil
}

func (r *RepositoryRepository) Update(_ context.Context, repository *repositories.Repository) error {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, repository))
	return nil
}

func (r *RepositoryRepository) ExecuteUpdate(_ context.Context, repository *repositories.Repository) error {
	err := r.txn.Insert("repositories", *repository)
	if err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	repository.ClearChanges()
	return nil
}

func (r *RepositoryRepository) Delete(_ context.Context, repository *repositories.Repository) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, repository))
	return nil
}

func (r *RepositoryRepository) ExecuteDelete(_ context.Context, repository *repositories.Repository) error {
	err := r.txn.Delete("repositories", repository)
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	return nil
}
