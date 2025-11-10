package repositories

import (
	"time"

	"github.com/google/uuid"
)

type BaseModel struct {
	id        uuid.UUID
	createdAt time.Time
	updatedAt time.Time
}

func NewBaseModel() BaseModel {
	return BaseModel{
		id:        uuid.New(),
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

func (b *BaseModel) GetId() uuid.UUID {
	return b.id
}

func (b *BaseModel) GetCreatedAt() time.Time {
	return b.createdAt
}

func (b *BaseModel) GetUpdatedAt() time.Time {
	return b.updatedAt
}
