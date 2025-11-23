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

type postgresProject struct {
	postgresBaseModel
	tenantId    uuid.UUID
	slug        string
	displayName string
	description *string
}

func (p *postgresProject) Map() *repositories.Project {
	return repositories.NewProjectFromDB(
		p.tenantId,
		p.slug,
		p.displayName,
		p.description,
		p.MapBase(),
	)
}

type projectRepository struct {
	tx *sql.Tx
}

func NewPostgresProjectRepository(tx *sql.Tx) repositories.ProjectRepository {
	return &projectRepository{
		tx: tx,
	}
}

func (r *projectRepository) selectQuery(filter *repositories.ProjectFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"projects.xmin",
		"projects.id",
		"projects.created_at",
		"projects.updated_at",
		"projects.tenant_id",
		"projects.slug",
		"projects.display_name",
		"projects.description",
	).From("projects")

	if filter.HasId() {
		s.Where(s.Equal("projects.id", filter.GetId()))
	}

	if filter.HasSlug() {
		s.Where(s.Equal("projects.slug", filter.GetSlug()))
	}

	if filter.HasTenantId() {
		s.Where(s.Equal("projects.tenant_id", filter.GetTenantId()))
	}

	return s
}

func (r *projectRepository) First(ctx context.Context, filter *repositories.ProjectFilter) (*repositories.Project, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	row := r.tx.QueryRowContext(ctx, query, args...)

	var project postgresProject
	err := row.Scan(&project.xmin, &project.id, &project.createdAt, &project.updatedAt, &project.tenantId, &project.slug, &project.displayName, &project.description)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return project.Map(), nil
}

func (r *projectRepository) Single(ctx context.Context, filter *repositories.ProjectFilter) (*repositories.Project, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiProjectNotFound
	}
	return result, nil
}

func (r *projectRepository) List(ctx context.Context, filter *repositories.ProjectFilter) ([]*repositories.Project, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.Build()
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var projects []*repositories.Project
	var totalCount int
	for rows.Next() {
		var project postgresProject
		err := rows.Scan(&project.xmin, &project.id, &project.createdAt, &project.updatedAt, &project.tenantId, &project.slug, &project.displayName, &project.description, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		projects = append(projects, project.Map())
	}

	return projects, totalCount, nil
}

func (r *projectRepository) Insert(ctx context.Context, project *repositories.Project) error {
	s := sqlbuilder.InsertInto("projects").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"tenant_id",
			"slug",
			"display_name",
			"description",
		).
		Values(
			project.GetId(),
			project.GetCreatedAt(),
			project.GetUpdatedAt(),
			project.GetTenantId(),
			project.GetSlug(),
			project.GetDisplayName(),
			project.GetDescription(),
		)

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}

func (r *projectRepository) Update(ctx context.Context, project *repositories.Project) error {
	panic("not yet implemented")
}

func (r *projectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("projects")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}
