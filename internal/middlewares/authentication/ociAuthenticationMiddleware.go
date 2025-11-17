package authentication

import (
	"net/http"

	"github.com/gorilla/mux"
)

func OciAuthenticationMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}

func getOciCurrentUser(w http.ResponseWriter, r *http.Request) (*CurrentUser, bool, error) {
	_, err := extractBearerToken(r, r.Header.Get("Authorization"))
	if err != nil {
		return nil, false, nil
	}

	return nil, false, nil
}
