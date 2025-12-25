package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/huandu/go-sqlbuilder"
	"github.com/lib/pq/hstore"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/logging"
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
	oidcRoleMapping     hstore.Hstore
}

func mapTenant(tenant *repositories.Tenant) *postgresTenant {
	oidcRoleMapping := hstore.Hstore{
		Map: make(map[string]sql.NullString),
	}

	for k, v := range tenant.GetOidcRoleMapping() {
		oidcRoleMapping.Map[k] = sql.NullString{String: v, Valid: true}
	}

	return &postgresTenant{
		postgresBaseModel:   mapBase(tenant.BaseModel),
		slug:                tenant.GetSlug(),
		displayName:         tenant.GetDisplayName(),
		oidcClient:          tenant.GetOidcClient(),
		oidcIssuer:          tenant.GetOidcIssuer(),
		oidcRoleClaim:       tenant.GetOidcRoleClaim(),
		oidcRoleClaimFormat: tenant.GetOidcRoleClaimFormat(),
		oidcRoleMapping:     oidcRoleMapping,
	}
}

func (t *postgresTenant) Map() *repositories.Tenant {
	oidcRoleMapping := make(map[string]string)
	for k, v := range t.oidcRoleMapping.Map {
		oidcRoleMapping[k] = v.String
	}

	return repositories.NewTenantFromDB(
		t.slug,
		t.displayName,
		repositories.NewTenantOidcConfig(
			t.oidcClient,
			t.oidcIssuer,
			t.oidcRoleClaim,
			t.oidcRoleClaimFormat,
			oidcRoleMapping,
		),
		t.MapBase(),
	)
}

func (t *postgresTenant) scan(row RowScanner) error {
	return row.Scan(
		&t.id,
		&t.createdAt,
		&t.updatedAt,
		&t.xmin,
		&t.slug,
		&t.displayName,
		&t.oidcClient,
		&t.oidcIssuer,
		&t.oidcRoleClaim,
	)
}

type TenantRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresTenantRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *TenantRepository {
	return &TenantRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *TenantRepository) selectQuery(filter *repositories.TenantFilter) *sqlbuilder.SelectBuilder {
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

func (r *TenantRepository) First(ctx context.Context, filter *repositories.TenantFilter) (*repositories.Tenant, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	tenant := &postgresTenant{}
	err := tenant.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return tenant.Map(), nil
}

func (r *TenantRepository) Single(ctx context.Context, filter *repositories.TenantFilter) (*repositories.Tenant, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiProjectNotFound
	}
	return result, nil
}

func (r *TenantRepository) List(ctx context.Context, filter *repositories.TenantFilter) ([]*repositories.Tenant, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var tenants []*repositories.Tenant
	var totalCount int
	for rows.Next() {
		tenant := &postgresTenant{}
		err := tenant.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		tenants = append(tenants, tenant.Map())
	}

	return tenants, totalCount, nil
}

func (r *TenantRepository) Insert(tenant *repositories.Tenant) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, tenant))
}

func (r *TenantRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, tenant *repositories.Tenant) error {
	mapped := mapTenant(tenant)

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
			mapped.id,
			mapped.createdAt,
			mapped.updatedAt,
			mapped.slug,
			mapped.displayName,
			mapped.oidcClient,
			mapped.oidcIssuer,
			mapped.oidcRoleClaim,
			mapped.oidcRoleClaimFormat,
			mapped.oidcRoleMapping,
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting tenant: %w", err)
	}

	tenant.SetVersion(xmin)
	tenant.ClearChanges()
	return nil
}

func (r *TenantRepository) Update(tenant *repositories.Tenant) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, tenant))
}

func (r *TenantRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, tenant *repositories.Tenant) error {
	if !tenant.HasChanges() {
		return nil
	}

	mapped := mapTenant(tenant)

	s := sqlbuilder.Update("tenants")
	s.Where(s.Equal("id", tenant.GetId()))
	s.Where(s.Equal("xmin", tenant.GetVersion()))

	for _, field := range tenant.GetChanges() {
		switch field {
		case repositories.TenantChangeOidcClient:
			s.SetMore(s.Assign("oidc_client", mapped.oidcClient))
		case repositories.TenantChangeOidcIssuer:
			s.SetMore(s.Assign("oidc_issuer", mapped.oidcIssuer))
		case repositories.TenantChangeOidcRoleMapping:
			s.SetMore(s.Assign("oidc_role_mapping", mapped.oidcRoleMapping))
		case repositories.TenantChangeOidcRoleClaim:
			s.SetMore(s.Assign("oidc_role_claim", mapped.oidcRoleClaim))
		case repositories.TenantChangeOidcRoleClaimFormat:
			s.SetMore(s.Assign("oidc_role_claim_format", mapped.oidcRoleClaimFormat))
		case repositories.TenantChangeDisplayName:
			s.SetMore(s.Assign("display_name", mapped.displayName))

		default:
			panic(fmt.Errorf("unknown tenant change: %d", field))
		}
	}

	s.Returning("xmin")
	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := tx.QueryRowContext(ctx, query, args...)

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

func (r *TenantRepository) Delete(tenant *repositories.Tenant) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, tenant))
}

func (r *TenantRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, tenant *repositories.Tenant) error {
	s := sqlbuilder.DeleteFrom("tenants")
	s.Where(s.Equal("id", tenant.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting tenant: %w", err)
	}

	return nil
}
