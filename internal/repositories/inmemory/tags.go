package inmemory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type tagRepository struct {
	txn *memdb.Txn
}

func NewInMemoryTagRepository(txn *memdb.Txn) repositories.TagRepository {
	return &tagRepository{
		txn: txn,
	}
}

func (r *tagRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.TagFilter) ([]*repositories.Tag, int, error) {
	var result []*repositories.Tag

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.Tag)

		if r.matches(&typed, filter) {
			if filter.GetIncludeManifestInfo() {
				manifestRepo := NewInMemoryManifestRepository(r.txn)
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

func (r *tagRepository) matches(tag *repositories.Tag, filter *repositories.TagFilter) bool {
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

func (r *tagRepository) First(_ context.Context, filter *repositories.TagFilter) (*repositories.Tag, error) {
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

func (r *tagRepository) Single(_ context.Context, filter *repositories.TagFilter) (*repositories.Tag, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiTagNotFound
	}
	return result, nil
}

func (r *tagRepository) List(_ context.Context, filter *repositories.TagFilter) ([]*repositories.Tag, int, error) {
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

func (r *tagRepository) Insert(_ context.Context, tag *repositories.Tag) error {
	err := r.txn.Insert("tags", *tag)
	if err != nil {
		return fmt.Errorf("failed to insert tag: %w", err)
	}

	return nil
}

func (r *tagRepository) Update(_ context.Context, tag *repositories.Tag) error {
	err := r.txn.Insert("tags", *tag)
	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}

	return nil
}

func (r *tagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	entry, err := r.First(ctx, repositories.NewTagFilter().ById(id))
	if err != nil {
		return fmt.Errorf("failed to get by id: %w", err)
	}
	if entry == nil {
		return nil
	}

	err = r.txn.Delete("tags", entry)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	return nil
}
