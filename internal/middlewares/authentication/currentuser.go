package authentication

import (
	"context"

	"github.com/google/uuid"
)

type CurrentUser struct {
	TenantId        uuid.UUID
	UserId          uuid.UUID
	Roles           []string
	IsAuthenticated bool
}

var CurrentUserContextKey = &CurrentUser{}

func ContextWithCurrentUser(ctx context.Context, user CurrentUser) context.Context {
	return context.WithValue(ctx, CurrentUserContextKey, user)
}

func GetCurrentUser(ctx context.Context) CurrentUser {
	value, ok := ctx.Value(CurrentUserContextKey).(CurrentUser)
	if !ok {
		panic("current user not found")
	}
	return value
}
