package inmemory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type manifestRepository struct {
	txn *memdb.Txn
}

func NewInMemoryManifestRepository(txn *memdb.Txn) repositories.ManifestRepository {
	return &manifestRepository{
		txn: txn,
	}
}

func (r *manifestRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.ManifestFilter) ([]*repositories.Manifest, int, error) {
	var result []*repositories.Manifest

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.Manifest)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count, nil
}

func (r *manifestRepository) matches(manifest *repositories.Manifest, filter *repositories.ManifestFilter) bool {
	if filter.HasId() {
		if manifest.GetId() != filter.GetId() {
			return false
		}
	}

	if filter.HasProjectId() {
		if manifest.GetProjectId() != filter.GetProjectId() {
			return false
		}
	}

	if filter.HasReference() {
		if manifest.GetReference() != filter.GetReference() {
			return false
		}
	}

	return true
}

func (r *manifestRepository) First(_ context.Context, filter *repositories.ManifestFilter) (*repositories.Manifest, error) {
	iterator, err := r.txn.Get("manifests", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get manifests: %w", err)
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

func (r *manifestRepository) Single(_ context.Context, filter *repositories.ManifestFilter) (*repositories.Manifest, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiManifestNotFound
	}
	return result, nil
}

func (r *manifestRepository) List(_ context.Context, filter *repositories.ManifestFilter) ([]*repositories.Manifest, int, error) {
	iterator, err := r.txn.Get("manifests", "id")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get manifests: %w", err)
	}

	result, count, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to apply filter: %w", err)
	}

	return result, count, nil
}

func (r *manifestRepository) Insert(_ context.Context, manifest *repositories.Manifest) error {
	err := r.txn.Insert("manifests", *manifest)
	if err != nil {
		return fmt.Errorf("failed to insert manifest: %w", err)
	}

	return nil
}

func (r *manifestRepository) Update(_ context.Context, manifest *repositories.Manifest) error {
	err := r.txn.Insert("manifests", *manifest)
	if err != nil {
		return fmt.Errorf("failed to update manifest: %w", err)
	}

	return nil
}

func (r *manifestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	entry, err := r.First(ctx, repositories.NewManifestFilter().ById(id))
	if err != nil {
		return fmt.Errorf("failed to get by id: %w", err)
	}
	if entry == nil {
		return nil
	}

	err = r.txn.Delete("manifests", entry)
	if err != nil {
		return fmt.Errorf("failed to delete manifest: %w", err)
	}

	return nil
}
