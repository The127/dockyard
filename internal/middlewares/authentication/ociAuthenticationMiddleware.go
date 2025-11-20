package authentication

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/The127/ioc"
	"github.com/The127/signr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func OciAuthenticationMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentUser, ok, err := getOciCurrentUser(r)
			if err != nil {
				ociError.HandleHttpError(w, r, err)
				return
			}

			if !ok {
				currentUser = &CurrentUser{
					TenantId:        uuid.Nil,
					UserId:          uuid.Nil,
					Roles:           []string{},
					IsAuthenticated: false,
				}
			}

			ctx := ContextWithCurrentUser(r.Context(), *currentUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getOciCurrentUser(r *http.Request) (*CurrentUser, bool, error) {
	bearerToken, err := extractBearerToken(r, r.Header.Get("Authorization"))
	if err != nil {
		return nil, false, nil
	}

	tenantSlug := strings.Split(r.Host, ".")[0]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	keyManager := ioc.GetDependency[signr.KeyManager](scope)

	signingKey, err := keyManager.
		GetGroup(fmt.Sprintf("jwt-signing-key:%s", tenantSlug)).
		GetKey("EdDSA")
	if err != nil {
		return nil, false, fmt.Errorf("getting signing key: %w", err)
	}

	token, err := jwt.Parse(bearerToken, func(token *jwt.Token) (interface{}, error) {
		return signingKey.PublicKey()
	}, jwt.WithAudience(tenantSlug), jwt.WithIssuer(config.C.Server.ExternalDomain), jwt.WithIssuedAt(), jwt.WithExpirationRequired())
	if err != nil {
		return nil, false, fmt.Errorf("parsing jwt: %w", err)
	}

	if !token.Valid {
		return nil, false, ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
	}

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		return nil, false, fmt.Errorf("getting transaction: %w", err)
	}

	claims := token.Claims.(jwt.MapClaims)

	tenant, err := tx.Tenants().First(ctx, repositories.NewTenantFilter().BySlug(claims["aud"].(string)))
	if err != nil {
		return nil, false, fmt.Errorf("failed to get tenant: %w", err)
	}
	if tenant == nil {
		return nil, false, ociError.NewOciError(ociError.Unauthorized).
			WithMessage("tenant not found").
			WithHttpCode(http.StatusUnauthorized)
	}

	subClaimString := claims["sub"].(string)
	userId, err := uuid.Parse(subClaimString)
	if err != nil {
		return nil, false, ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid user id").
			WithHttpCode(http.StatusUnauthorized)
	}

	return &CurrentUser{
		TenantId:        tenant.GetId(),
		UserId:          userId,
		Roles:           nil,
		IsAuthenticated: true,
	}, true, nil
}
