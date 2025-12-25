package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type RepositoryBlob struct {
	BaseModel

	repositoryId uuid.UUID
	blobId       uuid.UUID
}

func NewRepositoryBlob(repositoryId uuid.UUID, blobId uuid.UUID) *RepositoryBlob {
	return &RepositoryBlob{
		BaseModel:    NewBaseModel(),
		repositoryId: repositoryId,
		blobId:       blobId,
	}
}

func NewRepositoryBlobFromDB(repositoryId uuid.UUID, blobId uuid.UUID, base BaseModel) *RepositoryBlob {
	return &RepositoryBlob{
		BaseModel:    base,
		repositoryId: repositoryId,
		blobId:       blobId,
	}
}

func (r *RepositoryBlob) GetRepositoryId() uuid.UUID {
	return r.repositoryId
}

func (r *RepositoryBlob) GetBlobId() uuid.UUID {
	return r.blobId
}

type RepositoryBlobFilter struct {
	id           *uuid.UUID
	blobId       *uuid.UUID
	repositoryId *uuid.UUID
}

func NewRepositoryBlobFilter() *RepositoryBlobFilter {
	return &RepositoryBlobFilter{}
}

func (f *RepositoryBlobFilter) clone() *RepositoryBlobFilter {
	cloned := *f
	return &cloned
}

func (f *RepositoryBlobFilter) ById(id uuid.UUID) *RepositoryBlobFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *RepositoryBlobFilter) HasId() bool {
	return f.id != nil
}

func (f *RepositoryBlobFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *RepositoryBlobFilter) ByBlobId(id uuid.UUID) *RepositoryBlobFilter {
	cloned := f.clone()
	cloned.blobId = &id
	return cloned
}

func (f *RepositoryBlobFilter) HasBlobId() bool {
	return f.blobId != nil
}

func (f *RepositoryBlobFilter) GetBlobId() uuid.UUID {
	return pointer.DerefOrZero(f.blobId)
}

func (f *RepositoryBlobFilter) ByRepositoryId(id uuid.UUID) *RepositoryBlobFilter {
	cloned := f.clone()
	cloned.repositoryId = &id
	return cloned
}

func (f *RepositoryBlobFilter) HasRepositoryId() bool {
	return f.repositoryId != nil
}

func (f *RepositoryBlobFilter) GetRepositoryId() uuid.UUID {
	return pointer.DerefOrZero(f.repositoryId)
}

type RepositoryBlobRepository interface {
	Single(ctx context.Context, filter *RepositoryBlobFilter) (*RepositoryBlob, error)
	First(ctx context.Context, filter *RepositoryBlobFilter) (*RepositoryBlob, error)
	List(ctx context.Context, filter *RepositoryBlobFilter) ([]*RepositoryBlob, int, error)
	Insert(ctx context.Context, repositoryBlob *RepositoryBlob) error
	Delete(ctx context.Context, repositoryBlob *RepositoryBlob) error
}
