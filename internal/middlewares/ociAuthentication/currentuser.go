package ociAuthentication

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/middlewares"
)

type Access string

const (
	PushAccess Access = "push"
	PullAccess Access = "pull"
)

type CurrentUser struct {
	TenantId        uuid.UUID
	UserId          uuid.UUID
	IsAuthenticated bool
	Repository      *middlewares.OciRepositoryIdentifier
	Access          []Access
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
