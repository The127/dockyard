package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type Project struct {
	BaseModel

	tenantId uuid.UUID

	slug        string
	displayName string
}

func NewProject(tenantId uuid.UUID, slug string, displayName string) *Project {
	return &Project{
		BaseModel:   NewBaseModel(),
		tenantId:    tenantId,
		slug:        slug,
		displayName: displayName,
	}
}

func (p *Project) GetSlug() string {
	return p.slug
}

func (p *Project) GetDisplayName() string {
	return p.displayName
}

func (p *Project) SetDisplayName(displayName string) {
	p.displayName = displayName
}

func (p *Project) GetTenantId() uuid.UUID {
	return p.tenantId
}

type ProjectFilter struct {
	tenantId *uuid.UUID
	id       *uuid.UUID
	slug     *string
}

func NewProjectFilter() *ProjectFilter {
	return &ProjectFilter{}
}

func (f *ProjectFilter) clone() *ProjectFilter {
	cloned := *f
	return &cloned
}

func (f *ProjectFilter) ById(id uuid.UUID) *ProjectFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *ProjectFilter) HasId() bool {
	return f.id != nil
}

func (f *ProjectFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *ProjectFilter) ByTenantId(tenantId uuid.UUID) *ProjectFilter {
	cloned := f.clone()
	cloned.tenantId = &tenantId
	return cloned
}

func (f *ProjectFilter) HasTenantId() bool {
	return f.tenantId != nil
}

func (f *ProjectFilter) GetTenantId() uuid.UUID {
	return pointer.DerefOrZero(f.tenantId)
}

func (f *ProjectFilter) BySlug(slug string) *ProjectFilter {
	cloned := f.clone()
	cloned.slug = &slug
	return cloned
}

func (f *ProjectFilter) HasSlug() bool {
	return f.slug != nil
}

func (f *ProjectFilter) GetSlug() string {
	return pointer.DerefOrZero(f.slug)
}

type ProjectRepository interface {
	Single(ctx context.Context, filter *ProjectFilter) (*Project, error)
	First(ctx context.Context, filter *ProjectFilter) (*Project, error)
	List(ctx context.Context, filter *ProjectFilter) ([]*Project, int, error)
	Insert(ctx context.Context, project *Project) error
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}
