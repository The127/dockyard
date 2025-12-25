package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
)

type RepositoryAccessRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryRepositoryAccessRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) *RepositoryAccessRepository {
	return &RepositoryAccessRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *RepositoryAccessRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.RepositoryAccessFilter) ([]*repositories.RepositoryAccess, int) {
	var result []*repositories.RepositoryAccess

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.RepositoryAccess)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count
}

func (r *RepositoryAccessRepository) matches(repositoryAccess *repositories.RepositoryAccess, filter *repositories.RepositoryAccessFilter) bool {
	if filter.HasId() {
		if repositoryAccess.GetId() != filter.GetId() {
			return false
		}
	}

	if filter.HasUserId() {
		if repositoryAccess.GetUserId() != filter.GetUserId() {
			return false
		}
	}

	if filter.HasRepositoryId() {
		if repositoryAccess.GetRepositoryId() != filter.GetRepositoryId() {
			return false
		}
	}

	return true
}

func (r *RepositoryAccessRepository) First(_ context.Context, filter *repositories.RepositoryAccessFilter) (*repositories.RepositoryAccess, error) {
	iterator, err := r.txn.Get("repository_access", "id")
	if err != nil {
		return nil, err
	}

	result, _ := r.applyFilter(iterator, filter)

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *RepositoryAccessRepository) Insert(repositoryAccess *repositories.RepositoryAccess) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, repositoryAccess))
}

func (r *RepositoryAccessRepository) ExecuteInsert(tx *memdb.Txn, repositoryAccess *repositories.RepositoryAccess) error {
	err := tx.Insert("repository_access", *repositoryAccess)
	if err != nil {
		return fmt.Errorf("failed to insert repository access: %w", err)
	}

	repositoryAccess.ClearChanges()
	return nil
}

func (r *RepositoryAccessRepository) Update(repositoryAccess *repositories.RepositoryAccess) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, repositoryAccess))
}

func (r *RepositoryAccessRepository) ExecuteUpdate(tx *memdb.Txn, repositoryAccess *repositories.RepositoryAccess) error {
	err := tx.Insert("repository_access", *repositoryAccess)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}

	repositoryAccess.ClearChanges()
	return nil
}

func (r *RepositoryAccessRepository) Delete(repositoryAccess *repositories.RepositoryAccess) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, repositoryAccess))
}

func (r *RepositoryAccessRepository) ExecuteDelete(tx *memdb.Txn, repositoryAccess *repositories.RepositoryAccess) error {
	err := tx.Delete("repository_access", repositoryAccess)
	if err != nil {
		return fmt.Errorf("failed to delete repository access: %w", err)
	}

	return nil
}
