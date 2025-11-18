package ocihandlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func Root(w http.ResponseWriter, r *http.Request) {
	hostname := r.URL.Hostname()
	if hostname == config.C.Server.ExternalDomain {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	tenant := strings.Split(hostname, ".")[0]

	realm := fmt.Sprintf("%s/v2/token", config.C.Server.ExternalUrl)
	service := fmt.Sprintf("%s:%s", config.C.Server.ExternalDomain, tenant)

	wwwAuthenticateHeaderValue := fmt.Sprintf("Bearer realm=\"%s\",service=\"%s\"", realm, service)

	err := ociError.NewOciError(ociError.Unauthorized).
		WithMessage("user is not authenticated").
		WithHttpCode(401).
		WithHeader("WWW-Authenticate", wwwAuthenticateHeaderValue)

	ociError.HandleHttpError(w, r, err)
}
