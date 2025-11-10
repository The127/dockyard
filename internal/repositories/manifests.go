package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type Manifest struct {
	BaseModel

	projectId uuid.UUID

	reference string
	content   string
}

func (m *Manifest) GetProjectId() uuid.UUID {
	return m.projectId
}

func (m *Manifest) GetReference() string {
	return m.reference
}

func (m *Manifest) GetContent() string {
	return m.content
}

type ManifestFilter struct {
	id        *uuid.UUID
	projectId *uuid.UUID
	reference *string
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

func (f *ManifestFilter) ByProjectId(id uuid.UUID) *ManifestFilter {
	cloned := f.clone()
	cloned.projectId = &id
	return cloned
}

func (f *ManifestFilter) HasProjectId() bool {
	return f.projectId != nil
}

func (f *ManifestFilter) GetProjectId() uuid.UUID {
	return pointer.DerefOrZero(f.projectId)
}

func (f *ManifestFilter) ByReference(reference string) *ManifestFilter {
	cloned := f.clone()
	cloned.reference = &reference
	return cloned
}

func (f *ManifestFilter) HasReference() bool {
	return f.reference != nil
}

func (f *ManifestFilter) GetReference() string {
	return pointer.DerefOrZero(f.reference)
}

type ManifestRepository interface {
	Single(ctx context.Context, filter *ManifestFilter) (*Manifest, error)
	First(ctx context.Context, filter *ManifestFilter) (*Manifest, error)
	List(ctx context.Context, filter *ManifestFilter) ([]*Manifest, int, error)
	Insert(ctx context.Context, manifest *Manifest) error
	Update(ctx context.Context, manifest *Manifest) error
	Delete(ctx context.Context, id uuid.UUID) error
}
