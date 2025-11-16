package inmemory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type fileRepository struct {
	txn *memdb.Txn
}

func NewInMemoryFileRepository(txn *memdb.Txn) repositories.FileRepository {
	return &fileRepository{
		txn: txn,
	}
}

func (r *fileRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.FileFilter) ([]*repositories.File, int, error) {
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

	return result, count, nil
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

	result, _, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, err
	}

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

	result, count, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to apply filter: %w", err)
	}

	return result, count, nil
}

func (r *fileRepository) Insert(_ context.Context, file *repositories.File) error {
	err := r.txn.Insert("files", *file)
	if err != nil {
		return fmt.Errorf("failed to insert file: %w", err)
	}

	return nil
}

func (r *fileRepository) Update(_ context.Context, file *repositories.File) error {
	err := r.txn.Insert("files", *file)
	if err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}

	return nil
}

func (r *fileRepository) Delete(_ context.Context, id uuid.UUID) error {
	entry, err := r.First(context.Background(), repositories.NewFileFilter().ById(id))
	if err != nil {
		return fmt.Errorf("failed to get by id: %w", err)
	}
	if entry == nil {
		return nil
	}

	err = r.txn.Delete("files", entry)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}
