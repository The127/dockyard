package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/config"
)

type OciTenantSource string

const (
	// OciTenantSourcePath indicates that the tenant is specified in the URL path
	OciTenantSourcePath OciTenantSource = "path"

	// OciTenantSourceRoute indicates that the tenant is specified as the first route segment
	OciTenantSourceRoute OciTenantSource = "route"
)

type OciRepositoryIdentifier struct {
	TenantSlug     string
	ProjectSlug    string
	RepositorySlug string
}

type OciNameContextKey string

func OciNameMiddleware(tenantSource OciTenantSource) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)

			var tenant string
			switch tenantSource {
			case OciTenantSourcePath:
				tenant = vars["tenant"]

			case OciTenantSourceRoute:
				hostname := r.URL.Hostname()
				if hostname == config.C.Server.ExternalDomain {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				tenant = strings.Split(hostname, ".")[0]

			default:
				panic(fmt.Errorf("unsupported tenant source: %s", tenantSource))
			}

			repoIdentifier := OciRepositoryIdentifier{
				TenantSlug:     tenant,
				ProjectSlug:    vars["project"],
				RepositorySlug: vars["repository"],
			}

			r = r.WithContext(ContextWithRepoIdentifier(r.Context(), repoIdentifier))
			next.ServeHTTP(w, r)
		})
	}
}

func ContextWithRepoIdentifier(ctx context.Context, repoIdentifier OciRepositoryIdentifier) context.Context {
	return context.WithValue(ctx, OciNameContextKey("repoIdentifier"), repoIdentifier)
}

func GetRepoIdentifier(ctx context.Context) OciRepositoryIdentifier {
	return ctx.Value(OciNameContextKey("repoIdentifier")).(OciRepositoryIdentifier)
}
