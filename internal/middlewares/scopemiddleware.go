package middlewares

import (
	"context"
	"net/http"

	"github.com/The127/ioc"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/utils"
)

type scopeKeyType string

func ScopeMiddleware(root *ioc.DependencyProvider) mux.MiddlewareFunc {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scope := root.NewScope()
			defer utils.PanicOnError(scope.Close, "closing scope")

			r = r.WithContext(ContextWithScope(r.Context(), scope))
			handler.ServeHTTP(w, r)
		})
	}
}

func ContextWithScope(ctx context.Context, scope *ioc.DependencyProvider) context.Context {
	return context.WithValue(ctx, scopeKeyType("scope"), scope)
}

func GetScope(ctx context.Context) *ioc.DependencyProvider {
	return ctx.Value(scopeKeyType("scope")).(*ioc.DependencyProvider)
}
