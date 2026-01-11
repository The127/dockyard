package ociAuthentication

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
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func AuthenticationMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentUser, err := getOciCurrentUser(r)
			if err != nil {
				ociError.HandleHttpError(w, r, err)
				return
			}

			if currentUser == nil {
				currentUser = &CurrentUser{
					TenantId:        uuid.Nil,
					UserId:          uuid.Nil,
					IsAuthenticated: false,
				}
			}

			ctx := ContextWithCurrentUser(r.Context(), *currentUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getOciCurrentUser(r *http.Request) (*CurrentUser, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, nil
	}

	bearerToken := strings.TrimPrefix(authHeader, "Bearer ")

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	dbFactory := ioc.GetDependency[db.Factory](scope)
	dbContext, err := dbFactory.NewDbContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenantSlug := strings.Split(r.Host, ".")[0]
	tenant, err := dbContext.Tenants().First(ctx, repositories.NewTenantFilter().BySlug(tenantSlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	if tenant == nil {
		return nil, ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid tenant").
			WithHttpCode(http.StatusUnauthorized)
	}

	keyManager := ioc.GetDependency[signr.KeyManager](scope)

	signingKey, err := keyManager.
		GetGroup(fmt.Sprintf("jwt-signing-key:%s", tenantSlug)).
		GetKey("EdDSA")
	if err != nil {
		return nil, fmt.Errorf("getting signing key: %w", err)
	}

	token, err := jwt.Parse(
		bearerToken,
		func(token *jwt.Token) (interface{}, error) {
			return signingKey.PublicKey()
		},
		jwt.WithAudience(tenant.GetId().String()),
		jwt.WithIssuer(config.C.Server.ExternalDomain),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("parsing jwt: %w", err)
	}

	if !token.Valid {
		return nil, ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
	}

	claims := token.Claims.(jwt.MapClaims)
	audClaimString := claims["aud"].(string)
	tenantId, err := uuid.Parse(audClaimString)
	if err != nil {
		return nil, ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid tenant id").
			WithHttpCode(http.StatusUnauthorized)
	}

	subClaimString := claims["sub"].(string)
	userId, err := uuid.Parse(subClaimString)
	if err != nil {
		return nil, ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid user id").
			WithHttpCode(http.StatusUnauthorized)
	}

	var access []Access

	accessClaim, ok := claims["access"]
	if ok {
		accessClaimSlice, ok := accessClaim.([]any)
		if ok {
			for i := range accessClaimSlice {
				accessClaimString, ok := accessClaimSlice[i].(string)
				if ok {
					access = append(access, Access(accessClaimString))
				}
			}
		}
	}

	var repository *middlewares.OciRepositoryIdentifier
	repositoryClaim, ok := claims["repository"]
	if ok {
		repositoryClaimMap, ok := repositoryClaim.(map[string]any)
		if ok {
			repository = &middlewares.OciRepositoryIdentifier{
				TenantSlug:     repositoryClaimMap["tenant"].(string),
				ProjectSlug:    repositoryClaimMap["project"].(string),
				RepositorySlug: repositoryClaimMap["repository"].(string),
			}
		}
	}

	return &CurrentUser{
		TenantId:        tenantId,
		UserId:          userId,
		IsAuthenticated: true,
		Access:          access,
		Repository:      repository,
	}, nil
}
