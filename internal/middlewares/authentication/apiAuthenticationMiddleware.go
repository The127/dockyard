package authentication

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/utils/apiError"

	"github.com/coreos/go-oidc/v3/oidc"
)

func ApiAuthenticationMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			tenantSlug := vars["tenant"]

			currentUser, err := getApiCurrentUser(r, tenantSlug)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if currentUser == nil {
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

func getApiCurrentUser(r *http.Request, tenantSlug string) (*CurrentUser, error) {
	tokenStr, err := extractBearerToken(r, r.Header.Get("Authorization"))
	if err != nil {
		return nil, nil
	}

	// get the tenant
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	tenant, err := tx.Tenants().First(ctx, repositories.NewTenantFilter().BySlug(tenantSlug))
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	if tenant == nil {
		return nil, nil
	}

	// build oidc verifier
	provider, err := oidc.NewProvider(ctx, tenant.GetOidcIssuer())
	if err != nil {
		return nil, fmt.Errorf("failed to create oidc provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: tenant.GetOidcClient(),
	})

	// Verify token
	idToken, err := verifier.Verify(ctx, tokenStr)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", apiError.ErrApiUnauthorized)
	}

	// Extract roles claim (customizable per tenant)
	var claims map[string]interface{}
	err = idToken.Claims(&claims)
	if err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	var roles []string
	rawRoles, ok := claims[tenant.GetOidcRoleClaim()]
	if ok {
		switch tenant.GetOidcRoleClaimFormat() {
		case "array":
			rolesArray, ok := rawRoles.([]interface{})
			if ok {
				for i := range rolesArray {
					role, ok := rolesArray[i].(string)
					if ok {
						roles = append(roles, strings.TrimSpace(role))
					}
				}
			}

		case "space-separated":
			rolesString, ok := rawRoles.(string)
			if ok {
				roles = strings.Split(rolesString, " ")
			}

		case "comma-separated":
			rolesString, ok := rawRoles.(string)
			if ok {
				roles = strings.Split(rolesString, ",")
				for i, role := range roles {
					roles[i] = strings.TrimSpace(role)
				}
			}

		default:
			panic(fmt.Errorf("unsupported role claim format: %s", tenant.GetOidcRoleClaimFormat()))
		}
	}

	// apply tenant role mapping
	var mappedRoles []string
	tenantRoleMapping := tenant.GetOidcRoleMapping()
	for _, role := range roles {
		mappedRole, ok := tenantRoleMapping[role]
		if ok {
			mappedRoles = append(mappedRoles, mappedRole)
		}
	}

	// get or create the user
	user, err := tx.Users().First(ctx, repositories.NewUserFilter().BySubject(idToken.Subject))
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		user = repositories.NewUser(tenant.GetId(), idToken.Subject)

		emailClaim, ok := claims["email"].(string)
		if ok && emailClaim != "" {
			user.SetEmail(&emailClaim)
		}

		nameClaim, ok := claims["name"].(string)
		if ok && nameClaim != "" {
			user.SetDisplayName(&nameClaim)
		}

		err = tx.Users().Insert(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to insert user: %w", err)
		}
	}

	return &CurrentUser{
		TenantId: tenant.GetId(),
		UserId:   user.GetId(),
		Roles:    mappedRoles,
	}, nil
}
