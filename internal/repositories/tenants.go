package repositories

import (
	"context"

	"github.com/the127/dockyard/internal/utils/pointer"

	"github.com/google/uuid"
)

type Tenant struct {
	BaseModel

	slug        string
	displayName string
}

func (t *Tenant) GetSlug() string {
	return t.slug
}

func (t *Tenant) GetDisplayName() string {
	return t.displayName
}

func (t *Tenant) SetDisplayName(displayName string) {
	t.displayName = displayName
}

type TenantFilter struct {
	id   *uuid.UUID
	slug *string
}

func NewTenantFilter() *TenantFilter {
	return &TenantFilter{}
}

func (f *TenantFilter) clone() *TenantFilter {
	cloned := *f
	return &cloned
}

func (f *TenantFilter) ById(id uuid.UUID) *TenantFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *TenantFilter) HasId() bool {
	return f.id != nil
}

func (f *TenantFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *TenantFilter) BySlug(slug string) *TenantFilter {
	cloned := f.clone()
	cloned.slug = &slug
	return cloned
}

func (f *TenantFilter) HasSlug() bool {
	return f.slug != nil
}

func (f *TenantFilter) GetSlug() string {
	return pointer.DerefOrZero(f.slug)
}

type TenantRepository interface {
	Single(ctx context.Context, filter *TenantFilter) (*Tenant, error)
	First(ctx context.Context, filter *TenantFilter) (*Tenant, error)
	List(ctx context.Context, filter *TenantFilter) ([]*Tenant, int, error)
	Insert(ctx context.Context, tenant *Tenant) error
	Update(ctx context.Context, tenant *Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}
