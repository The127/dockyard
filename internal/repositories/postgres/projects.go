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

func mapProject(p *repositories.Project) *postgresProject {
	return &postgresProject{
		postgresBaseModel: mapBase(p.BaseModel),
		tenantId:          p.GetTenantId(),
		slug:              p.GetSlug(),
		displayName:       p.GetDisplayName(),
		description:       p.GetDescription(),
	}
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

func (p *postgresProject) scan(row RowScanner) error {
	return row.Scan(
		&p.id,
		&p.createdAt,
		&p.updatedAt,
		&p.xmin,
		&p.tenantId,
		&p.slug,
		&p.displayName,
		&p.description,
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
		"projects.id",
		"projects.created_at",
		"projects.updated_at",
		"projects.xmin",
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

	project := &postgresProject{}
	err := project.scan(row)
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
		project := &postgresProject{}
		err := project.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		projects = append(projects, project.Map())
	}

	return projects, totalCount, nil
}

func (r *ProjectRepository) Insert(project *repositories.Project) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, project))
}

func (r *ProjectRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, project *repositories.Project) error {
	mapped := mapProject(project)

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
			mapped.id,
			mapped.createdAt,
			mapped.updatedAt,
			mapped.tenantId,
			mapped.slug,
			mapped.displayName,
			mapped.description,
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting project: %w", err)
	}

	project.SetVersion(xmin)
	project.ClearChanges()
	return nil
}

func (r *ProjectRepository) Update(project *repositories.Project) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, project))
}

func (r *ProjectRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, project *repositories.Project) error {
	if !project.HasChanges() {
		return nil
	}

	mapped := mapProject(project)

	s := sqlbuilder.Update("projects")
	s.Where(s.Equal("id", project.GetId()))
	s.Where(s.Equal("xmin", project.GetVersion()))

	for _, field := range project.GetChanges() {
		switch field {
		case repositories.ProjectChangeDisplayName:
			s.SetMore(s.Assign("display_name", mapped.displayName))
		case repositories.ProjectChangeDescription:
			s.SetMore(s.Assign("description", mapped.description))
		default:
			panic(fmt.Errorf("unknown project change: %d", field))
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
		return fmt.Errorf("updating project: %w", apiError.ErrApiConcurrentUpdate)
	}

	if err != nil {
		return fmt.Errorf("updating project: %w", err)
	}

	project.SetVersion(xmin)
	project.ClearChanges()
	return nil
}

func (r *ProjectRepository) Delete(project *repositories.Project) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, project))
}

func (r *ProjectRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, project *repositories.Project) error {
	s := sqlbuilder.DeleteFrom("projects")
	s.Where(s.Equal("id", project.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting project: %w", err)
	}

	return nil
}
