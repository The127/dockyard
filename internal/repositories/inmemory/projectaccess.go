package inmemory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/repositories"
)

type projectAccessRepository struct {
	txn *memdb.Txn
}

func NewInMemoryProjectAccessRepository(txn *memdb.Txn) repositories.ProjectAccessRepository {
	return &projectAccessRepository{
		txn: txn,
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
	err := r.txn.Insert("project_access", *projectAccess)
	if err != nil {
		return fmt.Errorf("failed to insert project access: %w", err)
	}

	projectAccess.ClearChanges()
	return nil
}

func (r *projectAccessRepository) Update(_ context.Context, projectAccess *repositories.ProjectAccess) error {
	err := r.txn.Insert("project_access", *projectAccess)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}

	projectAccess.ClearChanges()
	return nil
}

func (r *projectAccessRepository) Delete(_ context.Context, id uuid.UUID) error {
	entry, err := r.First(context.Background(), repositories.NewProjectAccessFilter().ById(id))
	if err != nil {
		return fmt.Errorf("failed to get by id: %w", err)
	}
	if entry == nil {
		return nil
	}

	err = r.txn.Delete("project_access", entry)
	if err != nil {
		return fmt.Errorf("failed to delete project access: %w", err)
	}

	return nil
}
