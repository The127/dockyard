package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type User struct {
	BaseModel

	tenantId uuid.UUID
	subject  string

	displayName *string
	email       *string
}

func NewUser(tenantId uuid.UUID, subject string) *User {
	return &User{
		BaseModel:   NewBaseModel(),
		tenantId:    tenantId,
		subject:     subject,
		displayName: nil,
		email:       nil,
	}
}

func (u *User) GetTenantId() uuid.UUID {
	return u.tenantId
}

func (u *User) GetSubject() string {
	return u.subject
}

func (u *User) GetDisplayName() string {
	return pointer.DerefOrZero(u.displayName)
}

func (u *User) SetDisplayName(displayName *string) {
	u.displayName = displayName
}

func (u *User) GetEmail() string {
	return pointer.DerefOrZero(u.email)
}

func (u *User) SetEmail(email *string) {
	u.email = email
}

type UserFilter struct {
	tenantId *uuid.UUID
	id       *uuid.UUID
	subject  *string
}

func NewUserFilter() *UserFilter {
	return &UserFilter{}
}

func (f *UserFilter) clone() *UserFilter {
	cloned := *f
	return &cloned
}

func (f *UserFilter) ById(id uuid.UUID) *UserFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *UserFilter) HasId() bool {
	return f.id != nil
}

func (f *UserFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *UserFilter) ByTenantId(id uuid.UUID) *UserFilter {
	cloned := f.clone()
	cloned.tenantId = &id
	return cloned
}

func (f *UserFilter) HasTenantId() bool {
	return f.tenantId != nil
}

func (f *UserFilter) GetTenantId() uuid.UUID {
	return pointer.DerefOrZero(f.tenantId)
}

func (f *UserFilter) BySubject(subject string) *UserFilter {
	cloned := f.clone()
	cloned.subject = &subject
	return cloned
}

func (f *UserFilter) HasSubject() bool {
	return f.subject != nil
}

func (f *UserFilter) GetSubject() string {
	return pointer.DerefOrZero(f.subject)
}

type UserRepository interface {
	Single(ctx context.Context, filter *UserFilter) (*User, error)
	First(ctx context.Context, filter *UserFilter) (*User, error)
	List(ctx context.Context, filter *UserFilter) ([]*User, int, error)
	Insert(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}
