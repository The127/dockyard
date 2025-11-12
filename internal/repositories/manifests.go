package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type Manifest struct {
	BaseModel

	repositoryId uuid.UUID
	blobId       uuid.UUID

	digest string
}

func NewManifest(repositoryId uuid.UUID, blobId uuid.UUID, reference string) *Manifest {
	return &Manifest{
		BaseModel:    NewBaseModel(),
		repositoryId: repositoryId,
		blobId:       blobId,
		digest:       reference,
	}
}

func (m *Manifest) GetRepositoryId() uuid.UUID {
	return m.repositoryId
}

func (m *Manifest) GetDigest() string {
	return m.digest
}

func (m *Manifest) GetBlobId() uuid.UUID {
	return m.blobId
}

type ManifestFilter struct {
	id           *uuid.UUID
	repositoryId *uuid.UUID
	blobId       *uuid.UUID
	digest       *string
}

func NewManifestFilter() *ManifestFilter {
	return &ManifestFilter{}
}

func (f *ManifestFilter) clone() *ManifestFilter {
	cloned := *f
	return &cloned
}

func (f *ManifestFilter) ById(id uuid.UUID) *ManifestFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *ManifestFilter) HasId() bool {
	return f.id != nil
}

func (f *ManifestFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *ManifestFilter) ByBlobId(id uuid.UUID) *ManifestFilter {
	cloned := f.clone()
	cloned.blobId = &id
	return cloned
}

func (f *ManifestFilter) HasBlobId() bool {
	return f.blobId != nil
}

func (f *ManifestFilter) GetBlobId() uuid.UUID {
	return pointer.DerefOrZero(f.blobId)
}

func (f *ManifestFilter) ByRepositoryId(id uuid.UUID) *ManifestFilter {
	cloned := f.clone()
	cloned.repositoryId = &id
	return cloned
}

func (f *ManifestFilter) HasRepositoryId() bool {
	return f.repositoryId != nil
}

func (f *ManifestFilter) GetRepositoryId() uuid.UUID {
	return pointer.DerefOrZero(f.repositoryId)
}

func (f *ManifestFilter) ByDigest(digest string) *ManifestFilter {
	cloned := f.clone()
	cloned.digest = &digest
	return cloned
}

func (f *ManifestFilter) HasDigest() bool {
	return f.digest != nil
}

func (f *ManifestFilter) GetDigest() string {
	return pointer.DerefOrZero(f.digest)
}

type ManifestRepository interface {
	Single(ctx context.Context, filter *ManifestFilter) (*Manifest, error)
	First(ctx context.Context, filter *ManifestFilter) (*Manifest, error)
	List(ctx context.Context, filter *ManifestFilter) ([]*Manifest, int, error)
	Insert(ctx context.Context, manifest *Manifest) error
	Update(ctx context.Context, manifest *Manifest) error
	Delete(ctx context.Context, id uuid.UUID) error
}
