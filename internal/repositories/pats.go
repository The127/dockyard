package repositories

import (
	"context"
	"crypto/rand"
	"crypto/sha256"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type PatChange int

const (
	PatChangeDisplayName PatChange = iota
)

type Pat struct {
	BaseModel
	Changes[PatChange]

	userId       uuid.UUID
	displayName  string
	hashedSecret []byte
}

func NewPat(userId uuid.UUID, displayName string) (*Pat, []byte) {
	secret := make([]byte, 32)
	n, err := rand.Read(secret)
	if err != nil || n != len(secret) {
		panic("failed to generate random secret")
	}

	hashedSecret := sha256.New().Sum(secret)

	return &Pat{
		BaseModel:    NewBaseModel(),
		Changes:      NewChanges[PatChange](),
		userId:       userId,
		displayName:  displayName,
		hashedSecret: hashedSecret,
	}, secret
}

func NewPatFromDB(userId uuid.UUID, displayName string, hashedSecret []byte, base BaseModel) *Pat {
	return &Pat{
		BaseModel:    base,
		Changes:      NewChanges[PatChange](),
		userId:       userId,
		displayName:  displayName,
		hashedSecret: hashedSecret,
	}
}

func (p *Pat) GetUserId() uuid.UUID {
	return p.userId
}

func (p *Pat) GetHashedSecret() []byte {
	return p.hashedSecret
}

func (p *Pat) GetDisplayName() string {
	return p.displayName
}

func (p *Pat) SetDisplayName(displayName string) {
	if p.displayName == displayName {
		return
	}

	p.displayName = displayName
	p.trackChange(PatChangeDisplayName)
}

type PatFilter struct {
	id     *uuid.UUID
	userId *uuid.UUID
}

func NewPatFilter() *PatFilter {
	return &PatFilter{}
}

func (f *PatFilter) clone() *PatFilter {
	cloned := *f
	return &cloned
}

func (f *PatFilter) ById(id uuid.UUID) *PatFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *PatFilter) HasId() bool {
	return f.id != nil
}

func (f *PatFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *PatFilter) ByUserId(id uuid.UUID) *PatFilter {
	cloned := f.clone()
	cloned.userId = &id
	return cloned
}

func (f *PatFilter) HasUserId() bool {
	return f.userId != nil
}

func (f *PatFilter) GetUserId() uuid.UUID {
	return pointer.DerefOrZero(f.userId)
}

type PatRepository interface {
	Single(ctx context.Context, filter *PatFilter) (*Pat, error)
	First(ctx context.Context, filter *PatFilter) (*Pat, error)
	List(ctx context.Context, filter *PatFilter) ([]*Pat, int, error)
	Insert(ctx context.Context, entity *Pat) error
	Update(ctx context.Context, entity *Pat) error
	Delete(ctx context.Context, id uuid.UUID) error
}
