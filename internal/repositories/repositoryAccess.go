package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type RepositoryAccessRole string

const (
	RepositoryAccessRoleAdmin RepositoryAccessRole = "admin"
	RepositoryAccessRoleUser  RepositoryAccessRole = "user"
	RepositoryAccessRoleGuest RepositoryAccessRole = "reader"
)

func (r RepositoryAccessRole) AllowPush() bool {
	return r != RepositoryAccessRoleGuest
}

func (r RepositoryAccessRole) AllowPull() bool {
	// all roles can pull
	return true
}

type RepositoryAccess struct {
	BaseModel

	repositoryId uuid.UUID
	userId       uuid.UUID

	role RepositoryAccessRole
}

func NewRepositoryAccess(repositoryId uuid.UUID, userId uuid.UUID, role RepositoryAccessRole) *RepositoryAccess {
	return &RepositoryAccess{
		BaseModel:    NewBaseModel(),
		repositoryId: repositoryId,
		userId:       userId,
		role:         role,
	}
}

func (r *RepositoryAccess) GetRepositoryId() uuid.UUID {
	return r.repositoryId
}

func (r *RepositoryAccess) GetUserId() uuid.UUID {
	return r.userId
}

func (r *RepositoryAccess) GetRole() RepositoryAccessRole {
	return r.role
}

func (r *RepositoryAccess) SetRole(role RepositoryAccessRole) {
	r.role = role
}

type RepositoryAccessFilter struct {
	id           *uuid.UUID
	repositoryId *uuid.UUID
	userId       *uuid.UUID
}

func NewRepositoryAccessFilter() *RepositoryAccessFilter {
	return &RepositoryAccessFilter{}
}

func (f *RepositoryAccessFilter) clone() *RepositoryAccessFilter {
	cloned := *f
	return &cloned
}

func (f *RepositoryAccessFilter) ById(id uuid.UUID) *RepositoryAccessFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *RepositoryAccessFilter) HasId() bool {
	return f.id != nil
}

func (f *RepositoryAccessFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *RepositoryAccessFilter) ByRepositoryId(id uuid.UUID) *RepositoryAccessFilter {
	cloned := f.clone()
	cloned.repositoryId = &id
	return cloned
}

func (f *RepositoryAccessFilter) HasRepositoryId() bool {
	return f.repositoryId != nil
}

func (f *RepositoryAccessFilter) GetRepositoryId() uuid.UUID {
	return pointer.DerefOrZero(f.repositoryId)
}

func (f *RepositoryAccessFilter) ByUserId(id uuid.UUID) *RepositoryAccessFilter {
	cloned := f.clone()
	cloned.userId = &id
	return cloned
}

func (f *RepositoryAccessFilter) HasUserId() bool {
	return f.userId != nil
}

func (f *RepositoryAccessFilter) GetUserId() uuid.UUID {
	return pointer.DerefOrZero(f.userId)
}

type RepositoryAccessRepository interface {
	First(ctx context.Context, filter *RepositoryAccessFilter) (*RepositoryAccess, error)
	Insert(ctx context.Context, entity *RepositoryAccess) error
	Delete(ctx context.Context, id uuid.UUID) error
}
