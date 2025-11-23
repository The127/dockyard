package ocihandlers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/The127/ioc"
	"github.com/The127/signr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/middlewares/ociAuthentication"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/services/clock"
	"github.com/the127/dockyard/internal/utils/ociError"
)

type TokensResponse struct {
	Token     string `json:"token"`
	ExpiresAt int    `json:"expiresAt"`
	IssuedAt  string `json:"issuedAt"`
}

func Tokens(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	service := r.Form.Get("service")
	splitN := strings.SplitN(service, ":", 2)
	tenantSlug := splitN[1]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[services.DbService](scope)

	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	tenant, err := tx.Tenants().First(ctx, repositories.NewTenantFilter().BySlug(tenantSlug))
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}
	if tenant == nil {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid tenant").
			WithHttpCode(http.StatusUnauthorized)
		ociError.HandleHttpError(w, r, err)
		return
	}

	requestedScope := parseScopeFromRequest(r, tenantSlug)

	userId, err := getUserId(r, tenant)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	restrictedScope, err := restrictScope(ctx, tx, userId, requestedScope)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	keyManager := ioc.GetDependency[signr.KeyManager](scope)
	signingKey, err := keyManager.
		GetGroup(fmt.Sprintf("jwt-signing-key:%s", tenantSlug)).
		GetKey("EdDSA")
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	clockService := ioc.GetDependency[clock.Service](scope)
	now := clockService.Now()

	claims := map[string]any{
		"iss": config.C.Server.ExternalDomain,
		"sub": userId.String(),
		"aud": tenant.GetId().String(),
		"exp": jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
		"iat": jwt.NewNumericDate(now),
	}

	if restrictedScope != nil {
		claims["repository"] = restrictedScope.repository
		claims["access"] = restrictedScope.access
	}

	mapClaims := jwt.MapClaims(claims)

	s := NewJwtSigningMethod(signingKey)
	j, err := jwt.NewWithClaims(s, mapClaims).SignedString(s)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	response := TokensResponse{
		Token:     j,
		ExpiresAt: 10 * 60,
		IssuedAt:  now.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}
}

func checkAccessForUserAndRepository(
	ctx context.Context,
	tx database.Transaction,
	userId uuid.UUID,
	repository *repositories.Repository,
	accessType ociAuthentication.Access,
) (bool, error) {
	var repositoryAccessFilter *repositories.RepositoryAccessFilter
	var repositoryAccess *repositories.RepositoryAccess
	var err error

	if userId == uuid.Nil {
		if repository.GetIsPublic() && accessType == ociAuthentication.PullAccess {
			return true, nil
		} else {
			return false, nil
		}
	}

	repositoryAccessFilter = repositories.NewRepositoryAccessFilter().
		ByRepositoryId(repository.GetId()).
		ByUserId(userId)
	repositoryAccess, err = tx.RepositoryAccess().First(ctx, repositoryAccessFilter)
	if err != nil {
		return false, fmt.Errorf("getting repository access: %w", err)
	}

	if repositoryAccess == nil {
		return false, nil
	}

	if accessType == ociAuthentication.PushAccess && repositoryAccess.GetRole().AllowPush() {
		return true, nil
	}

	if accessType == ociAuthentication.PullAccess && repositoryAccess.GetRole().AllowPull() {
		return true, nil
	}

	return false, nil
}

func restrictScope(ctx context.Context, tx database.Transaction, userId uuid.UUID, scope *ociScope) (*ociScope, error) {
	if scope == nil {
		return nil, nil
	}

	_, _, repository, err := getRepositoryByIdentifier(ctx, tx, scope.repository)

	if err != nil {
		var ociErr *ociError.OciError
		if errors.As(err, &ociErr) && ociErr.Code == ociError.NameUnknown {
			return nil, nil
		}

		return nil, err
	}

	allowedAccesses := make([]ociAuthentication.Access, 0, len(scope.access))

	for _, access := range scope.access {
		ok, err := checkAccessForUserAndRepository(ctx, tx, userId, repository, access)
		if err != nil {
			return nil, err
		}

		if ok {
			allowedAccesses = append(allowedAccesses, access)
		}
	}

	if len(allowedAccesses) == 0 {
		return nil, nil
	}

	return &ociScope{
		repository: scope.repository,
		access:     allowedAccesses,
	}, nil
}

func parseScopeFromRequest(r *http.Request, tenantSlug string) *ociScope {
	scopeStr := r.Form.Get("scope")

	// repository:<reponame>:<accesslist>
	splitN := strings.SplitN(scopeStr, ":", 3)
	if len(splitN) != 3 || splitN[0] != "repository" {
		return nil
	}

	repository := splitN[1]

	accessStrs := strings.Split(splitN[2], ",")
	if len(accessStrs) == 0 {
		return nil
	}

	accesses := make([]ociAuthentication.Access, len(accessStrs))

	for _, accessStr := range accessStrs {
		if accessStr != string(ociAuthentication.PushAccess) && accessStr != string(ociAuthentication.PullAccess) {
			continue
		}

		accesses = append(accesses, ociAuthentication.Access(accessStr))
	}

	repositoryParts := strings.Split(repository, "/")
	if len(repositoryParts) != 2 {
		return nil
	}

	return &ociScope{
		repository: middlewares.OciRepositoryIdentifier{
			TenantSlug:     tenantSlug,
			ProjectSlug:    repositoryParts[0],
			RepositorySlug: repositoryParts[1],
		},
		access: accesses,
	}
}

type ociScope struct {
	repository middlewares.OciRepositoryIdentifier
	access     []ociAuthentication.Access
}

func getUserId(r *http.Request, tenant *repositories.Tenant) (uuid.UUID, error) {
	_, password, ok := r.BasicAuth()
	if !ok {
		return uuid.Nil, nil
	}

	if !strings.HasPrefix(password, "pat_") {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token, must start with 'pat_'").
			WithHttpCode(http.StatusUnauthorized)
		return uuid.Nil, err
	}

	patBytes, err := base64.RawURLEncoding.DecodeString(password[4:])
	if err != nil {
		return uuid.Nil, fmt.Errorf("decoding pat: %w", err)
	}

	uuidBytes := patBytes[:16]
	secretBytes := patBytes[16:]

	patId, err := uuid.FromBytes(uuidBytes)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parsing pat id: %w", err)
	}

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[services.DbService](scope)

	tx, err := dbService.GetTransaction()
	if err != nil {
		return uuid.Nil, fmt.Errorf("getting transaction: %w", err)
	}

	pat, err := tx.Pats().First(ctx, repositories.NewPatFilter().ById(patId))
	if err != nil {
		return uuid.Nil, fmt.Errorf("getting pat: %w", err)
	}
	if pat == nil {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
		return uuid.Nil, err
	}

	hashedSecret := sha256.New().Sum(secretBytes)
	if !slices.Equal(hashedSecret, pat.GetHashedSecret()) {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
		return uuid.Nil, err
	}

	user, err := tx.Users().First(ctx, repositories.NewUserFilter().ById(pat.GetUserId()))
	if err != nil {
		return uuid.Nil, fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
		return uuid.Nil, err
	}

	if tenant.GetId() != user.GetTenantId() {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
		return uuid.Nil, err
	}

	return user.GetId(), nil
}

type jwtSigningMethod struct {
	Key signr.SigningKey
}

func NewJwtSigningMethod(signingKey signr.SigningKey) jwt.SigningMethod {
	return &jwtSigningMethod{
		Key: signingKey,
	}
}

func (s *jwtSigningMethod) Alg() string { return s.Key.Algorithm() }

func (s *jwtSigningMethod) Sign(signingString string, _ any) ([]byte, error) {
	sig, err := s.Key.Sign([]byte(signingString))
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func (s *jwtSigningMethod) Verify(_ string, _ []byte, _ any) error {
	panic("this method should never be called")
}
