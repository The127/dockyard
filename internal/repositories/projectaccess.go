package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type ProjectAccessChange int

const (
	ProjectAccessChangeRole ProjectAccessChange = iota
)

type ProjectAccessRole string

const (
	ProjectAccessRoleAdmin ProjectAccessRole = "admin"
	ProjectAccessRoleUser  ProjectAccessRole = "user"
)

type ProjectAccess struct {
	BaseModel
	change.List[ProjectAccessChange]

	projectId uuid.UUID
	userId    uuid.UUID

	role ProjectAccessRole
}

func NewProjectAccess(projectId uuid.UUID, userId uuid.UUID, role ProjectAccessRole) *ProjectAccess {
	return &ProjectAccess{
		BaseModel: NewBaseModel(),
		List:      change.NewChanges[ProjectAccessChange](),
		projectId: projectId,
		userId:    userId,
		role:      role,
	}
}

func NewProjectAccessFromDB(projectId uuid.UUID, userId uuid.UUID, role ProjectAccessRole, base BaseModel) *ProjectAccess {
	return &ProjectAccess{
		BaseModel: base,
		List:      change.NewChanges[ProjectAccessChange](),
		projectId: projectId,
		userId:    userId,
		role:      role,
	}
}

func (p *ProjectAccess) GetProjectId() uuid.UUID {
	return p.projectId
}

func (p *ProjectAccess) GetUserId() uuid.UUID {
	return p.userId
}

func (p *ProjectAccess) GetRole() ProjectAccessRole {
	return p.role
}

func (p *ProjectAccess) SetRole(role ProjectAccessRole) {
	if p.role == role {
		return
	}

	p.role = role
	p.TrackChange(ProjectAccessChangeRole)
}

type ProjectAccessFilter struct {
	id        *uuid.UUID
	projectId *uuid.UUID
	userId    *uuid.UUID
}

func NewProjectAccessFilter() *ProjectAccessFilter {
	return &ProjectAccessFilter{}
}

func (f *ProjectAccessFilter) clone() *ProjectAccessFilter {
	cloned := *f
	return &cloned
}

func (f *ProjectAccessFilter) ById(id uuid.UUID) *ProjectAccessFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *ProjectAccessFilter) HasId() bool {
	return f.id != nil
}

func (f *ProjectAccessFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *ProjectAccessFilter) ByProjectId(id uuid.UUID) *ProjectAccessFilter {
	cloned := f.clone()
	cloned.projectId = &id
	return cloned
}

func (f *ProjectAccessFilter) HasProjectId() bool {
	return f.projectId != nil
}

func (f *ProjectAccessFilter) GetProjectId() uuid.UUID {
	return pointer.DerefOrZero(f.projectId)
}

func (f *ProjectAccessFilter) ByUserId(id uuid.UUID) *ProjectAccessFilter {
	cloned := f.clone()
	cloned.userId = &id
	return cloned
}

func (f *ProjectAccessFilter) HasUserId() bool {
	return f.userId != nil
}

func (f *ProjectAccessFilter) GetUserId() uuid.UUID {
	return pointer.DerefOrZero(f.userId)
}

type ProjectAccessRepository interface {
	First(ctx context.Context, filter *ProjectAccessFilter) (*ProjectAccess, error)
	Insert(ctx context.Context, entity *ProjectAccess) error
	Update(ctx context.Context, entity *ProjectAccess) error
	Delete(ctx context.Context, id uuid.UUID) error
}
