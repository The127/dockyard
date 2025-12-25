package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type TagRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryTagRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) *TagRepository {
	return &TagRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *TagRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.TagFilter) ([]*repositories.Tag, int, error) {
	var result []*repositories.Tag

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.Tag)

		if r.matches(&typed, filter) {
			if filter.GetIncludeManifestInfo() {
				manifestRepo := NewInMemoryManifestRepository(r.txn, r.changeTracker, -1)
				manifest, err := manifestRepo.Single(context.Background(), repositories.NewManifestFilter().ById(typed.GetRepositoryManifestId()))
				if err != nil {
					return nil, 0, fmt.Errorf("failed to get manifest for tag %s: %w", typed.GetId(), err)
				}

				typed.SetManifestInfo(repositories.TagManifestInfo{
					Digest: manifest.GetDigest(),
				})
			}

			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count, nil
}

func (r *TagRepository) matches(tag *repositories.Tag, filter *repositories.TagFilter) bool {
	if filter.HasId() {
		if tag.GetId() != filter.GetId() {
			return false
		}
	}

	if filter.HasRepositoryId() {
		if tag.GetRepositoryId() != filter.GetRepositoryId() {
			return false
		}
	}

	if filter.HasName() {
		if tag.GetName() != filter.GetName() {
			return false
		}
	}

	if filter.HasRepositoryManifestId() {
		if tag.GetRepositoryManifestId() != filter.GetRepositoryManifestId() {
			return false
		}
	}

	return true
}

func (r *TagRepository) First(_ context.Context, filter *repositories.TagFilter) (*repositories.Tag, error) {
	iterator, err := r.txn.Get("tags", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
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

func (r *TagRepository) Single(_ context.Context, filter *repositories.TagFilter) (*repositories.Tag, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiTagNotFound
	}
	return result, nil
}

func (r *TagRepository) List(_ context.Context, filter *repositories.TagFilter) ([]*repositories.Tag, int, error) {
	iterator, err := r.txn.Get("tags", "id")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tags: %w", err)
	}

	result, count, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to apply filter: %w", err)
	}

	return result, count, nil
}

func (r *TagRepository) Insert(tag *repositories.Tag) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, tag))
}

func (r *TagRepository) ExecuteInsert(tx *memdb.Txn, tag *repositories.Tag) error {
	err := tx.Insert("tags", *tag)
	if err != nil {
		return fmt.Errorf("failed to insert tag: %w", err)
	}

	return nil
}

func (r *TagRepository) Delete(tag *repositories.Tag) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, tag))
}

func (r *TagRepository) ExecuteDelete(tx *memdb.Txn, tag *repositories.Tag) error {
	err := tx.Delete("tags", tag)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	return nil
}
