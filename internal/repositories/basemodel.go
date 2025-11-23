package repositories

import (
	"time"

	"github.com/google/uuid"
)

type BaseModel struct {
	id        uuid.UUID
	createdAt time.Time
	updatedAt time.Time
	version   any
	changes   map[string]any
}

func NewBaseModel() BaseModel {
	return BaseModel{
		id:        uuid.New(),
		createdAt: time.Now(),
		updatedAt: time.Now(),
		version:   nil,
		changes:   make(map[string]any),
	}
}

func NewBaseModelFromDB(id uuid.UUID, createdAt time.Time, updatedAt time.Time, version any) BaseModel {
	return BaseModel{
		id:        id,
		createdAt: createdAt,
		updatedAt: updatedAt,
		version:   version,
		changes:   make(map[string]any),
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

func (b *BaseModel) GetVersion() any {
	return b.version
}

func (b *BaseModel) GetChanges() map[string]any {
	return b.changes
}

func (b *BaseModel) trackChange(key string, value any) {
	b.changes[key] = value
}
