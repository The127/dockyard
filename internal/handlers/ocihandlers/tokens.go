package ocihandlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	"github.com/the127/dockyard/internal/middlewares"
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
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
		ociError.HandleHttpError(w, r, err)
		return
	}

	userInfo, err := getUserInfo(r, tenant)
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

	claims := &jwt.MapClaims{
		"iss":    config.C.Server.ExternalDomain,
		"sub":    userInfo.Sub,
		"aud":    tenantSlug,
		"exp":    jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
		"iat":    jwt.NewNumericDate(now),
		"access": userInfo.Access,
	}

	s := NewJwtSigningMethod(signingKey)
	j, err := jwt.NewWithClaims(s, claims).SignedString(s)
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

type claimInfo struct {
	Sub    string
	Access *Access
}

type Access struct {
	Repository []string `json:"repository"`
}

func getUserInfo(r *http.Request, tenant *repositories.Tenant) (*claimInfo, error) {
	_, password, ok := r.BasicAuth()
	if !ok {
		return &claimInfo{
			Sub: uuid.Nil.String(),
		}, nil
	}

	if !strings.HasPrefix(password, "pat_") {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token, must start with 'pat_'").
			WithHttpCode(http.StatusUnauthorized)
		return nil, err
	}

	patBytes, err := base64.RawURLEncoding.DecodeString(password[4:])
	if err != nil {
		return nil, fmt.Errorf("decoding pat: %w", err)
	}

	uuidBytes := patBytes[:16]
	secretBytes := patBytes[16:]

	patId, err := uuid.FromBytes(uuidBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing pat id: %w", err)
	}

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[services.DbService](scope)

	tx, err := dbService.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	pat, err := tx.Pats().First(ctx, repositories.NewPatFilter().ById(patId))
	if err != nil {
		return nil, fmt.Errorf("getting pat: %w", err)
	}
	if pat == nil {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
		return nil, err
	}

	hashedSecret := sha256.New().Sum(secretBytes)
	if !slices.Equal(hashedSecret, pat.GetHashedSecret()) {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
		return nil, err
	}

	user, err := tx.Users().First(ctx, repositories.NewUserFilter().ById(pat.GetUserId()))
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
		return nil, err
	}

	if tenant.GetId() != user.GetTenantId() {
		err := ociError.NewOciError(ociError.Unauthorized).
			WithMessage("invalid token").
			WithHttpCode(http.StatusUnauthorized)
		return nil, err
	}

	return &claimInfo{
		Sub:    user.GetId().String(),
		Access: &Access{},
	}, nil
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
