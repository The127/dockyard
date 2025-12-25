package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type projectRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryProjectRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) *projectRepository {
	return &projectRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *projectRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.ProjectFilter) ([]*repositories.Project, int) {
	var result []*repositories.Project

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.Project)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count
}

func (r *projectRepository) matches(project *repositories.Project, filter *repositories.ProjectFilter) bool {
	if filter.HasSlug() {
		if project.GetSlug() != filter.GetSlug() {
			return false
		}
	}

	if filter.HasId() {
		if project.GetId() != filter.GetId() {
			return false
		}
	}

	if filter.HasTenantId() {
		if project.GetTenantId() != filter.GetTenantId() {
			return false
		}
	}

	return true
}

func (r *projectRepository) First(_ context.Context, filter *repositories.ProjectFilter) (*repositories.Project, error) {
	iterator, err := r.txn.Get("projects", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	result, _ := r.applyFilter(iterator, filter)

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *projectRepository) Single(ctx context.Context, filter *repositories.ProjectFilter) (*repositories.Project, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiProjectNotFound
	}
	return result, nil
}

func (r *projectRepository) List(_ context.Context, filter *repositories.ProjectFilter) ([]*repositories.Project, int, error) {
	iterator, err := r.txn.Get("projects", "id")
	if err != nil {
		return nil, 0, err
	}

	result, count := r.applyFilter(iterator, filter)

	return result, count, err
}

func (r *projectRepository) Insert(_ context.Context, project *repositories.Project) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, project))
	return nil
}

func (r *projectRepository) ExecuteInsert(_ context.Context, project *repositories.Project) error {
	err := r.txn.Insert("projects", *project)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}

	project.ClearChanges()
	return nil
}

func (r *projectRepository) Update(_ context.Context, project *repositories.Project) error {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, project))
	return nil
}

func (r *projectRepository) ExecuteUpdate(_ context.Context, project *repositories.Project) error {
	err := r.txn.Insert("projects", *project)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}

	project.ClearChanges()
	return nil
}

func (r *projectRepository) Delete(_ context.Context, project *repositories.Project) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, project))
	return nil
}

func (r *projectRepository) ExecuteDelete(_ context.Context, project *repositories.Project) error {
	err := r.txn.Delete("projects", project)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}
