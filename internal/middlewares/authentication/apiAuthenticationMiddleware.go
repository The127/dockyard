package authentication

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"

	"github.com/coreos/go-oidc/v3/oidc"
)

type CurrentUser struct {
	TenantId uuid.UUID
	UserId   uuid.UUID
	Roles    []string
}

var CurrentUserContextKey = &CurrentUser{}

func ApiAuthenticationMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			tenantSlug := vars["tenant"]

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// get the tenant
			ctx := r.Context()
			scope := middlewares.GetScope(ctx)

			dbService := ioc.GetDependency[services.DbService](scope)
			tx, err := dbService.GetTransaction()
			if err != nil {
				panic(fmt.Errorf("failed to get transaction: %w", err))
			}

			tenant, err := tx.Tenants().Single(ctx, repositories.NewTenantFilter().BySlug(tenantSlug))
			if err != nil {
				panic(fmt.Errorf("failed to get tenant: %w", err))
			}

			// build oidc verifier
			provider, err := oidc.NewProvider(ctx, tenant.GetOidcIssuer())
			if err != nil {
				panic(fmt.Errorf("failed to create oidc provider: %w", err))
			}

			verifier := provider.Verifier(&oidc.Config{
				ClientID: tenant.GetOidcClient(),
			})

			// Verify token
			idToken, err := verifier.Verify(ctx, tokenStr)
			if err != nil {
				http.Error(w, "failed to verify token", http.StatusUnauthorized)
				return
			}

			// Extract roles claim (customizable per tenant)
			var claims map[string]interface{}
			err = idToken.Claims(&claims)
			if err != nil {
				http.Error(w, "failed to parse token claims", http.StatusBadRequest)
				return
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

			// get or create user
			user, err := tx.Users().First(ctx, repositories.NewUserFilter().BySubject(idToken.Subject))
			if err != nil {
				panic(fmt.Errorf("failed to get user: %w", err))
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
					panic(fmt.Errorf("failed to insert user: %w", err))
				}
			}

			// set current user
			ctx = ContextWithCurrentUser(ctx, CurrentUser{
				TenantId: tenant.GetId(),
				UserId:   user.GetId(),
				Roles:    mappedRoles,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ContextWithCurrentUser(ctx context.Context, user CurrentUser) context.Context {
	return context.WithValue(ctx, CurrentUserContextKey, user)
}

func GetCurrentUser(ctx context.Context) CurrentUser {
	value, ok := ctx.Value(CurrentUserContextKey).(CurrentUser)
	if !ok {
		panic("current user not found")
	}
	return value
}
