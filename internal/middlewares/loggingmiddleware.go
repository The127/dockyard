package middlewares

import (
	"net/http"

	"github.com/the127/dockyard/internal/logging"

	"github.com/gorilla/mux"
)

func LoggingMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logging.Logger.Infof("API Request: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}
