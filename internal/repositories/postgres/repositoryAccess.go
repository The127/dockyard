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
)

type postgresRepositoryAccess struct {
	postgresBaseModel
	repositoryId uuid.UUID
	userId       uuid.UUID
	role         string
}

func (ra *postgresRepositoryAccess) Map() *repositories.RepositoryAccess {
	return repositories.NewRepositoryAccessFromDB(
		ra.repositoryId,
		ra.userId,
		repositories.RepositoryAccessRole(ra.role),
		ra.MapBase(),
	)
}

type repositoryAccessRepository struct {
	tx *sql.Tx
}

func NewPostgresRepositoryAccessRepository(tx *sql.Tx) repositories.RepositoryAccessRepository {
	return &repositoryAccessRepository{
		tx: tx,
	}
}

func (r *repositoryAccessRepository) selectQuery(filter *repositories.RepositoryAccessFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"repository_accesses.id",
		"repository_accesses.created_at",
		"repository_accesses.updated_at",
		"repository_accesses.repository_id",
		"repository_accesses.user_id",
		"repository_accesses.role",
	).From("repository_accesses")

	if filter.HasId() {
		s.Where(s.Equal("repository_accesses.id", filter.GetId()))
	}

	if filter.HasRepositoryId() {
		s.Where(s.Equal("repository_accesses.repository_id", filter.GetRepositoryId()))
	}

	return s
}

func (r *repositoryAccessRepository) First(ctx context.Context, filter *repositories.RepositoryAccessFilter) (*repositories.RepositoryAccess, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	row := r.tx.QueryRowContext(ctx, query, args...)

	var repositoryAccess postgresRepositoryAccess
	err := row.Scan(&repositoryAccess.id, &repositoryAccess.createdAt, &repositoryAccess.updatedAt, &repositoryAccess.repositoryId, &repositoryAccess.userId, &repositoryAccess.role)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return repositoryAccess.Map(), nil
}

func (r *repositoryAccessRepository) List(ctx context.Context, filter *repositories.RepositoryAccessFilter) ([]*repositories.RepositoryAccess, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.Build()
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var repositoryAccesses []*repositories.RepositoryAccess
	var totalCount int
	for rows.Next() {
		var repositoryAccess postgresRepositoryAccess
		err := rows.Scan(&repositoryAccess.id, &repositoryAccess.createdAt, &repositoryAccess.updatedAt, &repositoryAccess.repositoryId, &repositoryAccess.userId, &repositoryAccess.role, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		repositoryAccesses = append(repositoryAccesses, repositoryAccess.Map())
	}

	return repositoryAccesses, totalCount, nil
}

func (r *repositoryAccessRepository) Insert(ctx context.Context, repositoryAccess *repositories.RepositoryAccess) error {
	s := sqlbuilder.InsertInto("repository_access").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"repository_id",
			"user_id",
			"role",
		).
		Values(
			repositoryAccess.GetId(),
			repositoryAccess.GetCreatedAt(),
			repositoryAccess.GetUpdatedAt(),
			repositoryAccess.GetRepositoryId(),
			repositoryAccess.GetUserId(),
			repositoryAccess.GetRole(),
		)

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}

func (r *repositoryAccessRepository) Delete(ctx context.Context, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("repository_access")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}
