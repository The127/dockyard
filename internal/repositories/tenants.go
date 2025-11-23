package repositories

import (
	"context"
	"maps"

	"github.com/the127/dockyard/internal/utils/pointer"

	"github.com/google/uuid"
)

type TenantChange int

const (
	TenantChangeDisplayName TenantChange = iota
	TenantChangeOidcClient
	TenantChangeOidcIssuer
	TenantChangeOidcRoleClaim
	TenantChangeOidcRoleClaimFormat
	TenantChangeOidcRoleMapping
)

type Tenant struct {
	BaseModel
	Changes[TenantChange]

	slug        string
	displayName string

	oidcClient          string
	oidcIssuer          string
	oidcRoleClaim       string
	oidcRoleClaimFormat string
	oidcRoleMapping     map[string]string
}

type TenantOidcConfig struct {
	Client           string
	Issuer           string
	RoleClaim        string
	RoleClaimFormat  string
	RoleClaimMapping map[string]string
}

func NewTenantOidcConfig(
	client string,
	issuer string,
	roleClaim string,
	roleClaimFormat string,
	roleClaimMapping map[string]string,
) TenantOidcConfig {
	return TenantOidcConfig{
		Client:           client,
		Issuer:           issuer,
		RoleClaim:        roleClaim,
		RoleClaimFormat:  roleClaimFormat,
		RoleClaimMapping: roleClaimMapping,
	}
}

func NewTenant(slug string, displayName string, oidcConfig TenantOidcConfig) *Tenant {
	return &Tenant{
		BaseModel:           NewBaseModel(),
		slug:                slug,
		displayName:         displayName,
		oidcClient:          oidcConfig.Client,
		oidcIssuer:          oidcConfig.Issuer,
		oidcRoleClaim:       oidcConfig.RoleClaim,
		oidcRoleClaimFormat: oidcConfig.RoleClaimFormat,
		oidcRoleMapping:     oidcConfig.RoleClaimMapping,
	}
}

func NewTenantFromDB(slug string, displayName string, oidcConfig TenantOidcConfig, base BaseModel) *Tenant {
	return &Tenant{
		BaseModel:     base,
		slug:          slug,
		displayName:   displayName,
		oidcClient:    oidcConfig.Client,
		oidcIssuer:    oidcConfig.Issuer,
		oidcRoleClaim: oidcConfig.RoleClaim,
	}
}

func (t *Tenant) GetSlug() string {
	return t.slug
}

func (t *Tenant) GetDisplayName() string {
	return t.displayName
}

func (t *Tenant) SetDisplayName(displayName string) {
	if t.displayName == displayName {
		return
	}

	t.displayName = displayName
	t.trackChange(TenantChangeDisplayName)
}

func (t *Tenant) GetOidcClient() string {
	return t.oidcClient
}

func (t *Tenant) SetOidcClient(oidcClient string) {
	if t.oidcClient == oidcClient {
		return
	}

	t.oidcClient = oidcClient
	t.trackChange(TenantChangeOidcClient)
}

func (t *Tenant) GetOidcIssuer() string {
	return t.oidcIssuer
}

func (t *Tenant) SetOidcIssuer(oidcIssuer string) {
	if t.oidcIssuer == oidcIssuer {
		return
	}

	t.oidcIssuer = oidcIssuer
	t.trackChange(TenantChangeOidcIssuer)
}

func (t *Tenant) GetOidcRoleClaim() string {
	return t.oidcRoleClaim
}

func (t *Tenant) SetOidcRoleClaim(oidcRoleClaim string) {
	if t.oidcRoleClaim == oidcRoleClaim {
		return
	}

	t.oidcRoleClaim = oidcRoleClaim
	t.trackChange(TenantChangeOidcRoleClaim)
}

func (t *Tenant) GetOidcRoleClaimFormat() string {
	return t.oidcRoleClaimFormat
}

func (t *Tenant) SetOidcRoleClaimFormat(oidcRoleClaimFormat string) {
	if t.oidcRoleClaimFormat == oidcRoleClaimFormat {
		return
	}

	t.oidcRoleClaimFormat = oidcRoleClaimFormat
	t.trackChange(TenantChangeOidcRoleClaimFormat)
}

func (t *Tenant) GetOidcRoleMapping() map[string]string {
	return t.oidcRoleMapping
}

func (t *Tenant) SetOidcRoleMapping(oidcRoleMapping map[string]string) {
	if maps.Equal(t.oidcRoleMapping, oidcRoleMapping) {
		return
	}

	t.oidcRoleMapping = oidcRoleMapping
	t.trackChange(TenantChangeOidcRoleMapping)
}

type TenantFilter struct {
	id   *uuid.UUID
	slug *string
}

func NewTenantFilter() *TenantFilter {
	return &TenantFilter{}
}

func (f *TenantFilter) clone() *TenantFilter {
	cloned := *f
	return &cloned
}

func (f *TenantFilter) ById(id uuid.UUID) *TenantFilter {
	cloned := f.clone()
	cloned.id = &id
	return cloned
}

func (f *TenantFilter) HasId() bool {
	return f.id != nil
}

func (f *TenantFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.id)
}

func (f *TenantFilter) BySlug(slug string) *TenantFilter {
	cloned := f.clone()
	cloned.slug = &slug
	return cloned
}

func (f *TenantFilter) HasSlug() bool {
	return f.slug != nil
}

func (f *TenantFilter) GetSlug() string {
	return pointer.DerefOrZero(f.slug)
}

type TenantRepository interface {
	Single(ctx context.Context, filter *TenantFilter) (*Tenant, error)
	First(ctx context.Context, filter *TenantFilter) (*Tenant, error)
	List(ctx context.Context, filter *TenantFilter) ([]*Tenant, int, error)
	Insert(ctx context.Context, tenant *Tenant) error
	Update(ctx context.Context, tenant *Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}
