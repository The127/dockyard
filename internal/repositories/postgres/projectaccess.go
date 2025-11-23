package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils"
)

type postgresProjectAccess struct {
	id        uuid.UUID
	createdAt time.Time
	updatedAt time.Time
	projectId uuid.UUID
	userId    uuid.UUID
	role      string
}

func (b *postgresProjectAccess) Map() *repositories.ProjectAccess {
	return repositories.NewProjectAccessFromDB(
		b.projectId,
		b.userId,
		repositories.ProjectAccessRole(b.role),
		repositories.NewBaseModelFromDB(
			b.id,
			b.createdAt,
			b.updatedAt,
		),
	)
}

type projectAccessRepository struct {
	tx *sql.Tx
}

func NewPostgresProjectAccessRepository(tx *sql.Tx) repositories.ProjectAccessRepository {
	return &projectAccessRepository{
		tx: tx,
	}
}

func (r *projectAccessRepository) selectQuery(filter *repositories.ProjectAccessFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"project_accesses.id",
		"project_accesses.created_at",
		"project_accesses.updated_at",
		"project_accesses.project_id",
		"project_accesses.user_id",
		"project_accesses.role",
	).From("project_accesses")

	if filter.HasId() {
		s.Where(s.Equal("project_accesses.id", filter.GetId()))
	}

	if filter.HasProjectId() {
		s.Where(s.Equal("project_accesses.project_id", filter.GetProjectId()))
	}

	return s
}

func (r *projectAccessRepository) First(ctx context.Context, filter *repositories.ProjectAccessFilter) (*repositories.ProjectAccess, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	row := r.tx.QueryRowContext(ctx, query, args...)

	var projectAccess postgresProjectAccess
	err := row.Scan(&projectAccess.id, &projectAccess.createdAt, &projectAccess.updatedAt, &projectAccess.projectId, &projectAccess.userId, &projectAccess.role)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return projectAccess.Map(), nil
}

func (r *projectAccessRepository) List(ctx context.Context, filter *repositories.ProjectAccessFilter) ([]*repositories.ProjectAccess, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.Build()
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var projectAccesses []*repositories.ProjectAccess
	var totalCount int
	for rows.Next() {
		var projectAccess postgresProjectAccess
		err := rows.Scan(&projectAccess.id, &projectAccess.createdAt, &projectAccess.updatedAt, &projectAccess.projectId, &projectAccess.userId, &projectAccess.role, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		projectAccesses = append(projectAccesses, projectAccess.Map())
	}

	return projectAccesses, totalCount, nil
}

func (r *projectAccessRepository) Insert(ctx context.Context, projectAccess *repositories.ProjectAccess) error {
	s := sqlbuilder.InsertInto("project_access").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"project_id",
			"user_id",
			"role",
		).
		Values(
			projectAccess.GetId(),
			projectAccess.GetCreatedAt(),
			projectAccess.GetUpdatedAt(),
			projectAccess.GetProjectId(),
			projectAccess.GetUserId(),
			projectAccess.GetRole(),
		)

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}

func (r *projectAccessRepository) Delete(ctx context.Context, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("project_access")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}
