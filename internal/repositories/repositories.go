package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type Repository struct {
	BaseModel

	projectId uuid.UUID

	slug        string
	displayName string

	description  *string
	readmeFileId *uuid.UUID

	isPublic bool
}

func NewRepository(projectId uuid.UUID, slug string, displayName string) *Repository {
	return &Repository{
		BaseModel:   NewBaseModel(),
		projectId:   projectId,
		slug:        slug,
		displayName: displayName,
	}
}

func NewRepositoryFromDB(
	projectId uuid.UUID,
	slug string,
	displayName string,
	description *string,
	readmeFileId *uuid.UUID,
	isPublic bool,
	base BaseModel,
) *Repository {
	return &Repository{
		BaseModel:    base,
		projectId:    projectId,
		slug:         slug,
		displayName:  displayName,
		description:  description,
		readmeFileId: readmeFileId,
		isPublic:     isPublic,
	}
}

func (r *Repository) GetProjectId() uuid.UUID {
	return r.projectId
}

func (r *Repository) GetSlug() string {
	return r.slug
}

func (r *Repository) GetDisplayName() string {
	return r.displayName
}

func (r *Repository) GetDescription() *string {
	return r.description
}

func (r *Repository) SetDescription(description *string) {
	r.description = description
}

func (r *Repository) GetReadmeFileId() *uuid.UUID {
	return r.readmeFileId
}

func (r *Repository) SetReadmeFileId(readmeFileId *uuid.UUID) {
	r.readmeFileId = readmeFileId
}

func (r *Repository) GetIsPublic() bool {
	return r.isPublic
}

func (r *Repository) SetIsPublic(isPublic bool) {
	r.isPublic = isPublic
}

type RepositoryFilter struct {
	projectId *uuid.UUID
	id        *uuid.UUID
	slug      *string
}

func NewRepositoryFilter() *RepositoryFilter {
	return &RepositoryFilter{}
}

func (f *RepositoryFilter) clone() *RepositoryFilter {
	cloned := *f
	return &cloned
}

func (f *RepositoryFilter) ById(id uuid.UUID) *RepositoryFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *RepositoryFilter) HasId() bool {
	return f.id != nil
}

func (f *RepositoryFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *RepositoryFilter) ByProjectId(id uuid.UUID) *RepositoryFilter {
	cloned := f.clone()
	cloned.projectId = &id
	return cloned
}

func (f *RepositoryFilter) HasProjectId() bool {
	return f.projectId != nil
}

func (f *RepositoryFilter) GetProjectId() uuid.UUID {
	return pointer.DerefOrZero(f.projectId)
}

func (f *RepositoryFilter) BySlug(slug string) *RepositoryFilter {
	cloned := f.clone()
	cloned.slug = &slug
	return cloned
}

func (f *RepositoryFilter) HasSlug() bool {
	return f.slug != nil
}

func (f *RepositoryFilter) GetSlug() string {
	return pointer.DerefOrZero(f.slug)
}

type RepositoryRepository interface {
	Single(ctx context.Context, filter *RepositoryFilter) (*Repository, error)
	First(ctx context.Context, filter *RepositoryFilter) (*Repository, error)
	List(ctx context.Context, filter *RepositoryFilter) ([]*Repository, int, error)
	Insert(ctx context.Context, repository *Repository) error
	Update(ctx context.Context, repository *Repository) error
	Delete(ctx context.Context, id uuid.UUID) error
}
