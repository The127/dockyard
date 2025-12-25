package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type fileRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryFileRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) *fileRepository {
	return &fileRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *fileRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.FileFilter) ([]*repositories.File, int) {
	var result []*repositories.File

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.File)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count
}

func (r *fileRepository) matches(file *repositories.File, filter *repositories.FileFilter) bool {
	if filter.HasId() {
		if file.GetId() != filter.GetId() {
			return false
		}
	}

	if filter.HasDigest() {
		if file.GetDigest() != filter.GetDigest() {
			return false
		}
	}

	return true
}

func (r *fileRepository) First(_ context.Context, filter *repositories.FileFilter) (*repositories.File, error) {
	iterator, err := r.txn.Get("files", "id")
	if err != nil {
		return nil, err
	}

	result, _ := r.applyFilter(iterator, filter)

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *fileRepository) Single(_ context.Context, filter *repositories.FileFilter) (*repositories.File, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiFileNotFound
	}
	return result, nil
}

func (r *fileRepository) List(_ context.Context, filter *repositories.FileFilter) ([]*repositories.File, int, error) {
	iterator, err := r.txn.Get("files", "id")
	if err != nil {
		return nil, 0, err
	}

	result, count := r.applyFilter(iterator, filter)

	return result, count, nil
}

func (r *fileRepository) Insert(_ context.Context, file *repositories.File) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, file))
	return nil
}

func (r *fileRepository) ExecuteInsert(_ context.Context, file *repositories.File) error {
	err := r.txn.Insert("files", *file)
	if err != nil {
		return fmt.Errorf("failed to insert file: %w", err)
	}

	return nil
}

func (r *fileRepository) Delete(_ context.Context, file *repositories.File) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, file))
	return nil
}

func (r *fileRepository) ExecuteDelete(_ context.Context, file *repositories.File) error {
	err := r.txn.Delete("files", file)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}
