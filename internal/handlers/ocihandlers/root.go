package ocihandlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/middlewares/authentication"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func Root(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	currentUser := authentication.GetCurrentUser(ctx)
	if currentUser.IsAuthenticated {
		w.WriteHeader(http.StatusOK)
		return
	}

	tenant := strings.Split(r.Host, ".")[0]

	realm := fmt.Sprintf("%s/v2/token", config.C.Server.ExternalUrl)
	service := fmt.Sprintf("%s:%s", config.C.Server.ExternalDomain, tenant)

	wwwAuthenticateHeaderValue := fmt.Sprintf("Bearer realm=\"%s\",service=\"%s\"", realm, service)

	err := ociError.NewOciError(ociError.Unauthorized).
		WithMessage("user is not authenticated").
		WithHttpCode(401).
		WithHeader("WWW-Authenticate", wwwAuthenticateHeaderValue)

	ociError.HandleHttpError(w, r, err)
}
