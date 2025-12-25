package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type manifestRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryManifestRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) *manifestRepository {
	return &manifestRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *manifestRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.ManifestFilter) ([]*repositories.Manifest, int) {
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

	return result, count
}

func (r *manifestRepository) matches(manifest *repositories.Manifest, filter *repositories.ManifestFilter) bool {
	if filter.HasId() {
		if manifest.GetId() != filter.GetId() {
			return false
		}
	}

	if filter.HasRepositoryId() {
		if manifest.GetRepositoryId() != filter.GetRepositoryId() {
			return false
		}
	}

	if filter.HasDigest() {
		if manifest.GetDigest() != filter.GetDigest() {
			return false
		}
	}

	if filter.HasBlobId() {
		if manifest.GetBlobId() != filter.GetBlobId() {
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

	result, _ := r.applyFilter(iterator, filter)

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

	result, count := r.applyFilter(iterator, filter)

	return result, count, nil
}

func (r *manifestRepository) Insert(_ context.Context, manifest *repositories.Manifest) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, manifest))
	return nil
}

func (r *manifestRepository) ExecuteInsert(_ context.Context, manifest *repositories.Manifest) error {
	err := r.txn.Insert("manifests", *manifest)
	if err != nil {
		return fmt.Errorf("failed to insert manifest: %w", err)
	}

	return nil
}

func (r *manifestRepository) Update(_ context.Context, manifest *repositories.Manifest) error {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, manifest))
	return nil
}

func (r *manifestRepository) ExecuteUpdate(_ context.Context, manifest *repositories.Manifest) error {
	err := r.txn.Insert("manifests", *manifest)
	if err != nil {
		return fmt.Errorf("failed to update manifest: %w", err)
	}

	return nil
}

func (r *manifestRepository) Delete(_ context.Context, manifest *repositories.Manifest) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, manifest))
	return nil
}

func (r *manifestRepository) ExecuteDelete(_ context.Context, manifest *repositories.Manifest) error {
	err := r.txn.Delete("manifests", manifest)
	if err != nil {
		return fmt.Errorf("failed to delete manifest: %w", err)
	}

	return nil
}
