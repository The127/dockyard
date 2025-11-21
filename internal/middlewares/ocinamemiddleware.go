package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type OciRepositoryIdentifier struct {
	TenantSlug     string `json:"tenant"`
	ProjectSlug    string `json:"project"`
	RepositorySlug string `json:"repository"`
}

type OciNameContextKey string

func OciNameMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)

			tenant := strings.SplitN(r.Host, ".", 2)[0]

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
