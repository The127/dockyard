package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type TagManifestInfo struct {
	Digest string
}

type Tag struct {
	BaseModel

	repositoryId         uuid.UUID
	repositoryManifestId uuid.UUID

	name string

	manifestInfo *TagManifestInfo
}

func NewTag(repositoryId uuid.UUID, repositoryManifestId uuid.UUID, name string) *Tag {
	return &Tag{
		BaseModel:            NewBaseModel(),
		repositoryId:         repositoryId,
		repositoryManifestId: repositoryManifestId,
		name:                 name,
	}
}

func NewTagFromDB(repositoryId uuid.UUID, repositoryManifestId uuid.UUID, name string, manifestInfo *TagManifestInfo, base BaseModel) *Tag {
	return &Tag{
		BaseModel:            base,
		repositoryId:         repositoryId,
		repositoryManifestId: repositoryManifestId,
		name:                 name,
		manifestInfo:         manifestInfo,
	}
}

func (t *Tag) GetName() string {
	return t.name
}

func (t *Tag) GetRepositoryManifestId() uuid.UUID {
	return t.repositoryManifestId
}

func (t *Tag) GetRepositoryId() uuid.UUID {
	return t.repositoryId
}

func (t *Tag) GetManifestInfo() *TagManifestInfo {
	return t.manifestInfo
}

func (t *Tag) SetManifestInfo(manifestInfo TagManifestInfo) {
	t.manifestInfo = &manifestInfo
	// do not track changes on manifest info
}

type TagFilter struct {
	id                   *uuid.UUID
	repositoryId         *uuid.UUID
	repositoryManifestId *uuid.UUID
	name                 *string

	includeManifestInfo bool
}

func NewTagFilter() *TagFilter {
	return &TagFilter{}
}

func (f *TagFilter) clone() *TagFilter {
	cloned := *f
	return &cloned
}

func (f *TagFilter) ById(id uuid.UUID) *TagFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *TagFilter) HasId() bool {
	return f.id != nil
}

func (f *TagFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *TagFilter) ByRepositoryManifestId(id uuid.UUID) *TagFilter {
	cloned := f.clone()
	cloned.repositoryManifestId = &id
	return cloned
}

func (f *TagFilter) HasRepositoryManifestId() bool {
	return f.repositoryManifestId != nil
}

func (f *TagFilter) GetRepositoryManifestId() uuid.UUID {
	return pointer.DerefOrZero(f.repositoryManifestId)
}

func (f *TagFilter) ByRepositoryId(id uuid.UUID) *TagFilter {
	cloned := f.clone()
	cloned.repositoryId = &id
	return cloned
}

func (f *TagFilter) HasRepositoryId() bool {
	return f.repositoryId != nil
}

func (f *TagFilter) GetRepositoryId() uuid.UUID {
	return pointer.DerefOrZero(f.repositoryId)
}

func (f *TagFilter) ByName(name string) *TagFilter {
	cloned := f.clone()
	cloned.name = &name
	return cloned
}

func (f *TagFilter) HasName() bool {
	return f.name != nil
}

func (f *TagFilter) GetName() string {
	return pointer.DerefOrZero(f.name)
}

func (f *TagFilter) WithManifestInfo() *TagFilter {
	cloned := f.clone()
	cloned.includeManifestInfo = true
	return cloned
}

func (f *TagFilter) GetIncludeManifestInfo() bool {
	return f.includeManifestInfo
}

type TagRepository interface {
	Single(ctx context.Context, filter *TagFilter) (*Tag, error)
	First(ctx context.Context, filter *TagFilter) (*Tag, error)
	List(ctx context.Context, filter *TagFilter) ([]*Tag, int, error)
	Insert(ctx context.Context, tag *Tag) error
	Delete(ctx context.Context, tag *Tag) error
}
