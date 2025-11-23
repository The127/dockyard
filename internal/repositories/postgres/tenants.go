package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type postgresTenant struct {
	postgresBaseModel
	slug        string
	displayName string

	oidcClient          string
	oidcIssuer          string
	oidcRoleClaim       string
	oidcRoleClaimFormat string
	oidcRoleMapping     map[string]string
}

func (t *postgresTenant) Map() *repositories.Tenant {
	return repositories.NewTenantFromDB(
		t.slug,
		t.displayName,
		repositories.NewTenantOidcConfig(
			t.oidcClient,
			t.oidcIssuer,
			t.oidcRoleClaim,
			t.oidcRoleClaimFormat,
			t.oidcRoleMapping,
		),
		t.MapBase(),
	)
}

type tenantRepository struct {
	tx *sql.Tx
}

func NewPostgresTenantRepository(tx *sql.Tx) repositories.TenantRepository {
	return &tenantRepository{
		tx: tx,
	}
}

func (r *tenantRepository) selectQuery(filter *repositories.TenantFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"tenants.id",
		"tenants.created_at",
		"tenants.updated_at",
		"tenants.xmin",
		"tenants.slug",
		"tenants.display_name",
		"tenants.oidc_client",
		"tenants.oidc_issuer",
		"tenants.oidc_role_claim",
		"tenants.oidc_role_claim_format",
		"tenants.oidc_role_mapping",
	).From("tenants")

	if filter.HasId() {
		s.Where(s.Equal("tenants.id", filter.GetId()))
	}

	if filter.HasSlug() {
		s.Where(s.Equal("tenants.slug", filter.GetSlug()))
	}

	return s
}

func (r *tenantRepository) First(ctx context.Context, filter *repositories.TenantFilter) (*repositories.Tenant, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	row := r.tx.QueryRowContext(ctx, query, args...)

	var tenant postgresTenant
	err := row.Scan(&tenant.id, &tenant.createdAt, &tenant.updatedAt, &tenant.xmin, &tenant.slug, &tenant.displayName, &tenant.oidcClient, &tenant.oidcIssuer, &tenant.oidcRoleClaim, &tenant.oidcRoleClaimFormat, &tenant.oidcRoleMapping)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return tenant.Map(), nil
}

func (r *tenantRepository) Single(ctx context.Context, filter *repositories.TenantFilter) (*repositories.Tenant, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiProjectNotFound
	}
	return result, nil
}

func (r *tenantRepository) List(ctx context.Context, filter *repositories.TenantFilter) ([]*repositories.Tenant, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.Build()
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var tenants []*repositories.Tenant
	var totalCount int
	for rows.Next() {
		var tenant postgresTenant
		err := rows.Scan(&tenant.id, &tenant.createdAt, &tenant.updatedAt, &tenant.xmin, &tenant.slug, &tenant.displayName, &tenant.oidcClient, &tenant.oidcIssuer, &tenant.oidcRoleClaim, &tenant.oidcRoleClaimFormat, &tenant.oidcRoleMapping, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		tenants = append(tenants, tenant.Map())
	}

	return tenants, totalCount, nil
}

func (r *tenantRepository) Insert(ctx context.Context, tenant *repositories.Tenant) error {
	s := sqlbuilder.InsertInto("tenants").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"slug",
			"display_name",
			"oidc_client",
			"oidc_issuer",
			"oidc_role_claim",
			"oidc_role_claim_format",
			"oidc_role_mapping",
		).
		Values(
			tenant.GetId(),
			tenant.GetCreatedAt(),
			tenant.GetUpdatedAt(),
			tenant.GetSlug(),
			tenant.GetDisplayName(),
			tenant.GetOidcClient(),
			tenant.GetOidcIssuer(),
			tenant.GetOidcRoleClaim(),
			tenant.GetOidcRoleClaimFormat(),
			tenant.GetOidcRoleMapping(),
		)

	s.Returning("xmin")

	query, args := s.Build()
	row := r.tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting tenant: %w", err)
	}

	tenant.SetVersion(xmin)
	tenant.ClearChanges()
	return nil
}

func (r *tenantRepository) Update(ctx context.Context, tenant *repositories.Tenant) error {
	if !tenant.HasChanges() {
		return nil
	}

	s := sqlbuilder.Update("tenants")
	s.Where(s.Equal("id", tenant.GetId()))
	s.Where(s.Equal("xmin", tenant.GetVersion()))

	for _, field := range tenant.GetChanges() {
		switch field {
		case repositories.TenantChangeOidcIssuer:
			s.SetMore(s.Assign("oidc_issuer", tenant.GetOidcIssuer()))
		case repositories.TenantChangeOidcRoleMapping:
			s.SetMore(s.Assign("oidc_role_mapping", tenant.GetOidcRoleMapping()))
		case repositories.TenantChangeOidcClient:
			s.SetMore(s.Assign("oidc_client", tenant.GetOidcClient()))
		case repositories.TenantChangeOidcRoleClaim:
			s.SetMore(s.Assign("oidc_role_claim", tenant.GetOidcRoleClaim()))
		case repositories.TenantChangeOidcRoleClaimFormat:
			s.SetMore(s.Assign("oidc_role_claim_format", tenant.GetOidcRoleClaimFormat()))
		case repositories.TenantChangeDisplayName:
			s.SetMore(s.Assign("display_name", tenant.GetDisplayName()))

		default:
			panic(fmt.Errorf("unknown tenant change: %d", field))
		}
	}

	s.Returning("xmin")
	query, args := s.Build()
	row := r.tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if errors.Is(err, sql.ErrNoRows) {
		// no row was updated, which means the row was either already deleted or concurrently updated
		return fmt.Errorf("updating tenant: %w", apiError.ErrApiConcurrentUpdate)
	}

	if err != nil {
		return fmt.Errorf("updating tenant: %w", err)
	}

	tenant.SetVersion(xmin)
	tenant.ClearChanges()
	return nil
}

func (r *tenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("tenants")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting tenant: %w", err)
	}

	return nil
}
