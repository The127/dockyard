package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/lib/pq"
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

type patRepository struct {
	tx *sql.Tx
}

func NewPostgresPatRepository(tx *sql.Tx) repositories.PatRepository {
	return &patRepository{
		tx: tx,
	}
}

func (r *patRepository) selectQuery(filter *repositories.PatFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
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

func (r *patRepository) First(ctx context.Context, filter *repositories.PatFilter) (*repositories.Pat, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	row := r.tx.QueryRowContext(ctx, query, args...)

	var pat postgresPat
	err := row.Scan(&pat.id, &pat.createdAt, &pat.updatedAt, &pat.userId, &pat.displayName, pq.Array(&pat.hashedSecret))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return pat.Map(), nil
}

func (r *patRepository) Single(ctx context.Context, filter *repositories.PatFilter) (*repositories.Pat, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiPatNotFound
	}
	return result, nil
}

func (r *patRepository) List(ctx context.Context, filter *repositories.PatFilter) ([]*repositories.Pat, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.Build()
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var pats []*repositories.Pat
	var totalCount int
	for rows.Next() {
		var pat postgresPat
		err := rows.Scan(&pat.id, &pat.createdAt, &pat.updatedAt, &pat.userId, &pat.displayName, pq.Array(&pat.hashedSecret), &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		pats = append(pats, pat.Map())
	}

	return pats, totalCount, nil
}

func (r *patRepository) Insert(ctx context.Context, pat *repositories.Pat) error {
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
			pq.Array(pat.GetHashedSecret()),
		)

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}

func (r *patRepository) Delete(ctx context.Context, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("pats")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}
