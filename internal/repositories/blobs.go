package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type Blob struct {
	BaseModel

	digest string
}

func NewBlob(digest string) *Blob {
	return &Blob{
		BaseModel: NewBaseModel(),
		digest:    digest,
	}
}

func (b *Blob) GetDigest() string {
	return b.digest
}

type BlobFilter struct {
	id     *uuid.UUID
	digest *string
}

func NewBlobFilter() *BlobFilter {
	return &BlobFilter{}
}

func (f *BlobFilter) clone() *BlobFilter {
	cloned := *f
	return &cloned
}

func (f *BlobFilter) ById(id uuid.UUID) *BlobFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *BlobFilter) HasId() bool {
	return f.id != nil
}

func (f *BlobFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *BlobFilter) ByDigest(digest string) *BlobFilter {
	cloned := f.clone()
	cloned.digest = &digest
	return cloned
}

func (f *BlobFilter) HasDigest() bool {
	return f.digest != nil
}

func (f *BlobFilter) GetDigest() string {
	return pointer.DerefOrZero(f.digest)
}

type BlobRepository interface {
	Single(ctx context.Context, filter *BlobFilter) (*Blob, error)
	First(ctx context.Context, filter *BlobFilter) (*Blob, error)
	List(ctx context.Context, filter *BlobFilter) ([]*Blob, int, error)
	Insert(ctx context.Context, blob *Blob) error
	Update(ctx context.Context, blob *Blob) error
	Delete(ctx context.Context, id uuid.UUID) error
}
