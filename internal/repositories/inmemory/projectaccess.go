package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
)

type projectAccessRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryProjectAccessRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) *projectAccessRepository {
	return &projectAccessRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *projectAccessRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.ProjectAccessFilter) ([]*repositories.ProjectAccess, int) {
	var result []*repositories.ProjectAccess

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.ProjectAccess)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count
}

func (r *projectAccessRepository) matches(projectAccess *repositories.ProjectAccess, filter *repositories.ProjectAccessFilter) bool {
	if filter.HasId() {
		if projectAccess.GetId() != filter.GetId() {
			return false
		}
	}

	if filter.HasProjectId() {
		if projectAccess.GetProjectId() != filter.GetProjectId() {
			return false
		}
	}

	if filter.HasUserId() {
		if projectAccess.GetUserId() != filter.GetUserId() {
			return false
		}
	}

	return true
}

func (r *projectAccessRepository) First(_ context.Context, filter *repositories.ProjectAccessFilter) (*repositories.ProjectAccess, error) {
	iterator, err := r.txn.Get("project_access", "id")
	if err != nil {
		return nil, err
	}

	result, _ := r.applyFilter(iterator, filter)

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *projectAccessRepository) Insert(_ context.Context, projectAccess *repositories.ProjectAccess) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, projectAccess))
	return nil
}

func (r *projectAccessRepository) ExecuteInsert(_ context.Context, projectAccess *repositories.ProjectAccess) error {
	err := r.txn.Insert("project_access", *projectAccess)
	if err != nil {
		return fmt.Errorf("failed to insert project access: %w", err)
	}

	projectAccess.ClearChanges()
	return nil
}

func (r *projectAccessRepository) Update(_ context.Context, projectAccess *repositories.ProjectAccess) error {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, projectAccess))
	return nil
}

func (r *projectAccessRepository) ExecuteUpdate(_ context.Context, projectAccess *repositories.ProjectAccess) error {
	err := r.txn.Insert("project_access", *projectAccess)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}

	projectAccess.ClearChanges()
	return nil
}

func (r *projectAccessRepository) Delete(_ context.Context, projectAccess *repositories.ProjectAccess) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, projectAccess))
	return nil
}

func (r *projectAccessRepository) ExecuteDelete(_ context.Context, projectAccess *repositories.ProjectAccess) error {
	err := r.txn.Delete("project_access", projectAccess)
	if err != nil {
		return fmt.Errorf("failed to delete project access: %w", err)
	}

	return nil
}
