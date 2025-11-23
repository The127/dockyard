package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/repositories"
)

type postgresBaseModel struct {
	id        uuid.UUID
	createdAt time.Time
	updatedAt time.Time
	xmin      uint
}

func (b *postgresBaseModel) MapBase() repositories.BaseModel {
	return repositories.NewBaseModelFromDB(b.id, b.createdAt, b.updatedAt, b.xmin)
}

func newPostgresBaseModel(baseModel repositories.BaseModel) postgresBaseModel {
	return postgresBaseModel{
		id:        baseModel.GetId(),
		createdAt: baseModel.GetCreatedAt(),
		updatedAt: baseModel.GetUpdatedAt(),
		xmin:      baseModel.GetVersion().(uint),
	}
}
