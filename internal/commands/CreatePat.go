package commands

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
)

type CreatePat struct {
	UserId      uuid.UUID
	DisplayName string
}

type CreatePatResponse struct {
	Token string
}

func HandleCreatePat(ctx context.Context, command CreatePat) (*CreatePatResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[services.DbService](scope)

	tx, err := dbService.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	pat, secret := repositories.NewPat(command.UserId, command.DisplayName)
	err = tx.Pats().Insert(ctx, pat)
	if err != nil {
		return nil, fmt.Errorf("inserting pat: %w", err)
	}

	secretBase64 := base64.RawURLEncoding.EncodeToString(secret)
	token := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", pat.GetId(), secretBase64)))

	return &CreatePatResponse{
		Token: token,
	}, nil
}
