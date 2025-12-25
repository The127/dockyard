package queries

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
)

type ListUsers struct {
	TenantSlug *string
}

type ListUsersResponse PagedResponse[ListUsersResponseItem]

type ListUsersResponseItem struct {
	Id          uuid.UUID
	Subject     string
	DisplayName *string
	Email       *string
}

func HandleListUsers(ctx context.Context, query ListUsers) (*ListUsersResponse, error) {
	scope := middlewares.GetScope(ctx)

	dbFactory := ioc.GetDependency[db.Factory](scope)
	dbContext, err := dbFactory.NewDbContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	userFilter := repositories.NewUserFilter()

	if query.TenantSlug != nil {
		tenantFilter := repositories.NewTenantFilter().BySlug(*query.TenantSlug)
		tenant, err := dbContext.Tenants().Single(ctx, tenantFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to get tenant: %w", err)
		}

		userFilter = userFilter.ByTenantId(tenant.GetId())
	}

	users, _, err := dbContext.Users().List(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	items := make([]ListUsersResponseItem, len(users))
	for i, user := range users {
		items[i] = ListUsersResponseItem{
			Id:          user.GetId(),
			Subject:     user.GetSubject(),
			DisplayName: user.GetDisplayName(),
			Email:       user.GetEmail(),
		}
	}

	return &ListUsersResponse{
		Items: items,
	}, nil
}
