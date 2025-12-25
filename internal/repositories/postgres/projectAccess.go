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

type postgresProjectAccess struct {
	postgresBaseModel
	projectId uuid.UUID
	userId    uuid.UUID
	role      string
}

func (pa *postgresProjectAccess) Map() *repositories.ProjectAccess {
	return repositories.NewProjectAccessFromDB(
		pa.projectId,
		pa.userId,
		repositories.ProjectAccessRole(pa.role),
		pa.MapBase(),
	)
}

type ProjectAccessRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresProjectAccessRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *ProjectAccessRepository {
	return &ProjectAccessRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ProjectAccessRepository) selectQuery(filter *repositories.ProjectAccessFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"project_accesses.xmin",
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

func (r *ProjectAccessRepository) First(ctx context.Context, filter *repositories.ProjectAccessFilter) (*repositories.ProjectAccess, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	var projectAccess postgresProjectAccess
	err := row.Scan(&projectAccess.xmin, &projectAccess.id, &projectAccess.createdAt, &projectAccess.updatedAt, &projectAccess.projectId, &projectAccess.userId, &projectAccess.role)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return projectAccess.Map(), nil
}

func (r *ProjectAccessRepository) List(ctx context.Context, filter *repositories.ProjectAccessFilter) ([]*repositories.ProjectAccess, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var projectAccesses []*repositories.ProjectAccess
	var totalCount int
	for rows.Next() {
		var projectAccess postgresProjectAccess
		err := rows.Scan(&projectAccess.xmin, &projectAccess.id, &projectAccess.createdAt, &projectAccess.updatedAt, &projectAccess.projectId, &projectAccess.userId, &projectAccess.role, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		projectAccesses = append(projectAccesses, projectAccess.Map())
	}

	return projectAccesses, totalCount, nil
}

func (r *ProjectAccessRepository) Insert(projectAccess *repositories.ProjectAccess) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, projectAccess))
}

func (r *ProjectAccessRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, projectAccess *repositories.ProjectAccess) error {
	s := sqlbuilder.InsertInto("project_accesses").
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

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting project access: %w", err)
	}

	projectAccess.SetVersion(xmin)
	projectAccess.ClearChanges()
	return nil
}

func (r *ProjectAccessRepository) Update(projectAccess *repositories.ProjectAccess) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, projectAccess))
}

func (r *ProjectAccessRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, projectAccess *repositories.ProjectAccess) error {
	if !projectAccess.HasChanges() {
		return nil
	}

	s := sqlbuilder.Update("project_accesses")
	s.Where(s.Equal("id", projectAccess.GetId()))
	s.Where(s.Equal("xmin", projectAccess.GetVersion()))

	for _, field := range projectAccess.GetChanges() {
		switch field {
		case repositories.ProjectAccessChangeRole:
			s.SetMore(s.Assign("role", projectAccess.GetRole()))
		default:
			panic(fmt.Errorf("unknown project access change: %d", field))
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
		return fmt.Errorf("updating project access: %w", apiError.ErrApiConcurrentUpdate)
	}

	if err != nil {
		return fmt.Errorf("updating project access: %w", err)
	}

	projectAccess.SetVersion(xmin)
	projectAccess.ClearChanges()
	return nil
}

func (r *ProjectAccessRepository) Delete(projectAccess *repositories.ProjectAccess) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, projectAccess))
}

func (r *ProjectAccessRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, projectAccess *repositories.ProjectAccess) error {
	s := sqlbuilder.DeleteFrom("project_accesses")
	s.Where(s.Equal("id", projectAccess.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting project access: %w", err)
	}

	return nil
}
