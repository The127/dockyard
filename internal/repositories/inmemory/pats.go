package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type PatRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryPatRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) *PatRepository {
	return &PatRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *PatRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.PatFilter) ([]*repositories.Pat, int) {
	var result []*repositories.Pat

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.Pat)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count
}

func (r *PatRepository) matches(pat *repositories.Pat, filter *repositories.PatFilter) bool {
	if filter.HasId() {
		if pat.GetId() != filter.GetId() {
			return false
		}
	}

	if filter.HasUserId() {
		if pat.GetUserId() != filter.GetUserId() {
			return false
		}
	}

	return true
}

func (r *PatRepository) First(_ context.Context, filter *repositories.PatFilter) (*repositories.Pat, error) {
	iterator, err := r.txn.Get("pats", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get pats: %w", err)
	}

	result, _ := r.applyFilter(iterator, filter)

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *PatRepository) Single(_ context.Context, filter *repositories.PatFilter) (*repositories.Pat, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, apiError.ErrApiPatNotFound
	}

	return result, nil
}

func (r *PatRepository) List(_ context.Context, filter *repositories.PatFilter) ([]*repositories.Pat, int, error) {
	iterator, err := r.txn.Get("pats", "id")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get pats: %w", err)
	}

	pats, count := r.applyFilter(iterator, filter)

	return pats, count, nil
}

func (r *PatRepository) Insert(pat *repositories.Pat) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, pat))
}

func (r *PatRepository) ExecuteInsert(tx *memdb.Txn, pat *repositories.Pat) error {
	err := tx.Insert("pats", *pat)
	if err != nil {
		return fmt.Errorf("failed to insert pat: %w", err)
	}

	pat.ClearChanges()
	return nil
}

func (r *PatRepository) Update(pat *repositories.Pat) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, pat))
}

func (r *PatRepository) ExecuteUpdate(tx *memdb.Txn, pat *repositories.Pat) error {
	err := tx.Insert("pats", *pat)
	if err != nil {
		return fmt.Errorf("failed to update pat: %w", err)
	}

	pat.ClearChanges()
	return nil
}

func (r *PatRepository) Delete(pat *repositories.Pat) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, pat))
}

func (r *PatRepository) ExecuteDelete(tx *memdb.Txn, pat *repositories.Pat) error {
	err := tx.Delete("pats", pat)
	if err != nil {
		return fmt.Errorf("failed to delete pat: %w", err)
	}

	return nil
}
