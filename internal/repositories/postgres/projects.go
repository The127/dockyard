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

type ProjectRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresProjectRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *ProjectRepository {
	return &ProjectRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ProjectRepository) selectQuery(filter *repositories.ProjectFilter) *sqlbuilder.SelectBuilder {
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

func (r *ProjectRepository) First(ctx context.Context, filter *repositories.ProjectFilter) (*repositories.Project, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

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

func (r *ProjectRepository) Single(ctx context.Context, filter *repositories.ProjectFilter) (*repositories.Project, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiProjectNotFound
	}
	return result, nil
}

func (r *ProjectRepository) List(ctx context.Context, filter *repositories.ProjectFilter) ([]*repositories.Project, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.db.QueryContext(ctx, query, args...)
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

func (r *ProjectRepository) Insert(ctx context.Context, project *repositories.Project) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, project))
	return nil
}

func (r *ProjectRepository) ExecuteInsert(ctx context.Context, project *repositories.Project) error {
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

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting project: %w", err)
	}

	project.SetVersion(xmin)
	project.ClearChanges()
	return nil
}

func (r *ProjectRepository) Update(ctx context.Context, project *repositories.Project) error {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, project))
	return nil
}

func (r *ProjectRepository) ExecuteUpdate(ctx context.Context, project *repositories.Project) error {
	if !project.HasChanges() {
		return nil
	}

	s := sqlbuilder.Update("projects")
	s.Where(s.Equal("id", project.GetId()))
	s.Where(s.Equal("xmin", project.GetVersion()))

	for _, field := range project.GetChanges() {
		switch field {
		case repositories.ProjectChangeDisplayName:
			s.SetMore(s.Assign("display_name", project.GetDisplayName()))
		case repositories.ProjectChangeDescription:
			s.SetMore(s.Assign("description", project.GetDescription()))
		default:
			panic(fmt.Errorf("unknown project change: %d", field))
		}
	}

	s.Returning("xmin")
	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if errors.Is(err, sql.ErrNoRows) {
		// no row was updated, which means the row was either already deleted or concurrently updated
		return fmt.Errorf("updating project: %w", apiError.ErrApiConcurrentUpdate)
	}

	if err != nil {
		return fmt.Errorf("updating project: %w", err)
	}

	project.SetVersion(xmin)
	project.ClearChanges()
	return nil
}

func (r *ProjectRepository) Delete(ctx context.Context, project *repositories.Project) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, project))
	return nil
}

func (r *ProjectRepository) ExecuteDelete(ctx context.Context, project *repositories.Project) error {
	s := sqlbuilder.DeleteFrom("projects")
	s.Where(s.Equal("id", project.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting project: %w", err)
	}

	return nil
}
