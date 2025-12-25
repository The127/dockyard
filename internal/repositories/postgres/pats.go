package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/logging"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type postgresPat struct {
	postgresBaseModel
	userId       uuid.UUID
	displayName  string
	hashedSecret []byte
}

func (p *postgresPat) Map() *repositories.Pat {
	return repositories.NewPatFromDB(
		p.userId,
		p.displayName,
		p.hashedSecret,
		p.MapBase(),
	)
}

type PatRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresPatRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *PatRepository {
	return &PatRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *PatRepository) selectQuery(filter *repositories.PatFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"pats.xmin",
		"pats.id",
		"pats.created_at",
		"pats.updated_at",
		"pats.user_id",
		"pats.display_name",
		"pats.hashed_secret",
	).From("pats")

	if filter.HasId() {
		s.Where(s.Equal("pats.id", filter.GetId()))
	}

	if filter.HasUserId() {
		s.Where(s.Equal("pats.user_id", filter.GetUserId()))
	}

	return s
}

func (r *PatRepository) First(ctx context.Context, filter *repositories.PatFilter) (*repositories.Pat, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	var pat postgresPat
	err := row.Scan(&pat.xmin, &pat.id, &pat.createdAt, &pat.updatedAt, &pat.userId, &pat.displayName, &pat.hashedSecret)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return pat.Map(), nil
}

func (r *PatRepository) Single(ctx context.Context, filter *repositories.PatFilter) (*repositories.Pat, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiPatNotFound
	}
	return result, nil
}

func (r *PatRepository) List(ctx context.Context, filter *repositories.PatFilter) ([]*repositories.Pat, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var pats []*repositories.Pat
	var totalCount int
	for rows.Next() {
		var pat postgresPat
		err := rows.Scan(&pat.xmin, &pat.id, &pat.createdAt, &pat.updatedAt, &pat.userId, &pat.displayName, &pat.hashedSecret, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		pats = append(pats, pat.Map())
	}

	return pats, totalCount, nil
}

func (r *PatRepository) Insert(ctx context.Context, pat *repositories.Pat) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, pat))
	return nil
}

func (r *PatRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, pat *repositories.Pat) error {
	s := sqlbuilder.InsertInto("pats").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"user_id",
			"display_name",
			"hashed_secret",
		).
		Values(
			pat.GetId(),
			pat.GetCreatedAt(),
			pat.GetUpdatedAt(),
			pat.GetUserId(),
			pat.GetDisplayName(),
			pat.GetHashedSecret(),
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting pat: %w", err)
	}

	pat.SetVersion(xmin)
	return nil
}

func (r *PatRepository) Update(ctx context.Context, pat *repositories.Pat) error {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, pat))
	return nil
}

func (r *PatRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, pat *repositories.Pat) error {
	if !pat.HasChanges() {
		return nil
	}

	s := sqlbuilder.Update("pats")
	s.Where(s.Equal("id", pat.GetId()))
	s.Where(s.Equal("xmin", pat.GetVersion()))

	for _, field := range pat.GetChanges() {
		switch field {
		case repositories.PatChangeDisplayName:
			s.SetMore(s.Assign("display_name", pat.GetDisplayName()))
		default:
			panic(fmt.Errorf("unknown pat change: %d", field))
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
		return fmt.Errorf("updating pat: %w", apiError.ErrApiConcurrentUpdate)
	}

	if err != nil {
		return fmt.Errorf("updating pat: %w", err)
	}

	pat.SetVersion(xmin)
	pat.ClearChanges()
	return nil
}

func (r *PatRepository) Delete(ctx context.Context, pat *repositories.Pat) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, pat))
	return nil
}

func (r *PatRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, pat *repositories.Pat) error {
	s := sqlbuilder.DeleteFrom("pats")
	s.Where(s.Equal("id", pat.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting pat: %w", err)
	}

	return nil
}
