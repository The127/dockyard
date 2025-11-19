package queries

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
)

type ListPats struct {
	UserId uuid.UUID
}

type ListPatsResponse PagedResponse[ListPatsResponseItem]

type ListPatsResponseItem struct {
	Id          uuid.UUID
	DisplayName string
}

func HandleListPats(ctx context.Context, query ListPats) (*ListPatsResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[services.DbService](scope)

	tx, err := dbService.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	patFilter := repositories.NewPatFilter().ByUserId(query.UserId)
	pats, _, err := tx.Pats().List(ctx, patFilter)
	if err != nil {
		return nil, fmt.Errorf("listing pats: %w", err)
	}

	items := make([]ListPatsResponseItem, len(pats))

	for i, pat := range pats {
		items[i] = ListPatsResponseItem{
			Id:          pat.GetId(),
			DisplayName: pat.GetDisplayName(),
		}
	}

	return &ListPatsResponse{
		Items: items,
	}, nil
}
