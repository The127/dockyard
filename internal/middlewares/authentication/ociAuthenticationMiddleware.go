package authentication

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func OciAuthenticationMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentUser, ok, err := getOciCurrentUser(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if !ok {
				currentUser = &CurrentUser{
					TenantId: uuid.Nil,
					UserId:   uuid.Nil,
					Roles:    []string{},
				}
			}

			ctx := ContextWithCurrentUser(r.Context(), *currentUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getOciCurrentUser(r *http.Request) (*CurrentUser, bool, error) {
	_, err := extractBearerToken(r, r.Header.Get("Authorization"))
	if err != nil {
		return nil, false, nil
	}

	return nil, false, nil
}
