package commands

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/The127/go-clock"
	"github.com/The127/ioc"
	"github.com/google/uuid"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
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
	dbContext := ioc.GetDependency[db.Context](scope)

	clockService := ioc.GetDependency[clock.Service](scope)
	var displayName = command.DisplayName
	if displayName == "" {
		now := clockService.Now().Format("2006-01-02 15:04:05")
		displayName = fmt.Sprintf("Dockyard PAT %s", now)
	}

	pat, secret := repositories.NewPat(command.UserId, displayName)
	dbContext.Pats().Insert(pat)

	tokenBytes := make([]byte, 16+len(secret)) // 16 bytes of uuid + length of secret

	idBytes, err := pat.GetId().MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshalling pat id: %w", err)
	}

	copy(tokenBytes, idBytes)
	copy(tokenBytes[16:], secret)

	tokenBase64 := base64.RawURLEncoding.EncodeToString(tokenBytes)

	return &CreatePatResponse{
		Token: fmt.Sprintf("pat_%s", tokenBase64),
	}, nil
}
