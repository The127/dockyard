package repositories

import (
	"context"
	"crypto/rand"
	"crypto/sha256"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type Pat struct {
	BaseModel

	userId       uuid.UUID
	hashedSecret []byte
}

func NewPat(userId uuid.UUID) (*Pat, []byte) {
	secret := make([]byte, 32)
	n, err := rand.Read(secret)
	if err != nil || n != len(secret) {
		panic("failed to generate random secret")
	}

	hashedSecret := sha256.New().Sum(secret)

	return &Pat{
		BaseModel:    NewBaseModel(),
		userId:       userId,
		hashedSecret: hashedSecret,
	}, secret
}

func (p *Pat) GetUserId() uuid.UUID {
	return p.userId
}

func (p *Pat) GetHashedSecret() []byte {
	return p.hashedSecret
}

type PatFilter struct {
	id *uuid.UUID
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

type PatRepository interface {
	Single(ctx context.Context, filter *PatFilter) (*Pat, error)
	First(ctx context.Context, filter *PatFilter) (*Pat, error)
	Insert(ctx context.Context, entity *Pat) error
	Delete(ctx context.Context, id uuid.UUID) error
}
