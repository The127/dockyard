package inmemory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type projectRepository struct {
	txn *memdb.Txn
}

func NewInMemoryProjectRepository(txn *memdb.Txn) repositories.ProjectRepository {
	return &projectRepository{
		txn: txn,
	}
}

func (r *projectRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.ProjectFilter) ([]*repositories.Project, int, error) {
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

	return result, count, nil
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

	result, _, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to apply filter: %w", err)
	}

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

	result, count, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to apply filter: %w", err)
	}

	return result, count, err
}

func (r *projectRepository) Insert(_ context.Context, project *repositories.Project) error {
	err := r.txn.Insert("projects", *project)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}

	return nil
}

func (r *projectRepository) Update(_ context.Context, project *repositories.Project) error {
	err := r.txn.Insert("projects", *project)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}

	return nil
}

func (r *projectRepository) Delete(_ context.Context, id uuid.UUID) error {
	entry, err := r.First(context.Background(), repositories.NewProjectFilter().ById(id))
	if err != nil {
		return fmt.Errorf("failed to get by id: %w", err)
	}
	if entry == nil {
		return nil
	}

	err = r.txn.Delete("projects", entry)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}
