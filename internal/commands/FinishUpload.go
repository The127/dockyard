package commands

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
)

type FinishUpload struct {
	RepositoryId   uuid.UUID
	ComputedDigest string
	Size           int64
}

type FinishUploadResponse struct{}

func HandleFinishUpload(ctx context.Context, command FinishUpload) (*FinishUploadResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[db.Context](scope)

	blob, err := getOrCreateBlob(ctx, dbContext, command.ComputedDigest, command.Size)
	if err != nil {
		return nil, err
	}

	dbContext.RepositoryBlobs().Insert(repositories.NewRepositoryBlob(command.RepositoryId, blob.GetId()))

	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("saving changes: %w", err)
	}

	return &FinishUploadResponse{}, nil
}
