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
}

func NewBaseModel() BaseModel {
	return BaseModel{
		id:        uuid.New(),
		createdAt: time.Now(),
		updatedAt: time.Now(),
		version:   nil,
	}
}

func NewBaseModelFromDB(id uuid.UUID, createdAt time.Time, updatedAt time.Time, version any) BaseModel {
	return BaseModel{
		id:        id,
		createdAt: createdAt,
		updatedAt: updatedAt,
		version:   version,
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

// SetVersion is used to update the version of the model.
// This is used to prevent concurrent updates.
// This function should only be called by the repositories.
func (b *BaseModel) SetVersion(version any) {
	b.version = version
}
