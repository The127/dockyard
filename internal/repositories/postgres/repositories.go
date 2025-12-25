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

type postgresRepository struct {
	postgresBaseModel
	projectId    uuid.UUID
	slug         string
	displayName  string
	description  *string
	readmeFileId *uuid.UUID
	isPublic     bool
}

func mapRepository(r *repositories.Repository) *postgresRepository {
	return &postgresRepository{
		postgresBaseModel: mapBase(r.BaseModel),
		projectId:         r.GetProjectId(),
		slug:              r.GetSlug(),
		displayName:       r.GetDisplayName(),
		description:       r.GetDescription(),
		readmeFileId:      r.GetReadmeFileId(),
		isPublic:          r.GetIsPublic(),
	}
}

func (r *postgresRepository) Map() *repositories.Repository {
	return repositories.NewRepositoryFromDB(
		r.projectId,
		r.slug,
		r.displayName,
		r.description,
		r.readmeFileId,
		r.isPublic,
		r.MapBase(),
	)
}

func (r *postgresRepository) scan(row RowScanner) error {
	return row.Scan(
		&r.id,
		&r.createdAt,
		&r.updatedAt,
		&r.xmin,
		&r.projectId,
		&r.slug,
		&r.displayName,
		&r.description,
		&r.readmeFileId,
		&r.isPublic,
	)
}

type RepositoryRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresRepositoryRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *RepositoryRepository {
	return &RepositoryRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *RepositoryRepository) selectQuery(filter *repositories.RepositoryFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"repositories.id",
		"repositories.created_at",
		"repositories.updated_at",
		"repositories.xmin",
		"repositories.project_id",
		"repositories.slug",
		"repositories.display_name",
		"repositories.description",
		"repositories.readme_file_id",
		"repositories.is_public",
	).From("repositories")

	if filter.HasId() {
		s.Where(s.Equal("repositories.id", filter.GetId()))
	}

	if filter.HasSlug() {
		s.Where(s.Equal("repositories.slug", filter.GetSlug()))
	}

	if filter.HasProjectId() {
		s.Where(s.Equal("repositories.project_id", filter.GetProjectId()))
	}

	return s
}

func (r *RepositoryRepository) First(ctx context.Context, filter *repositories.RepositoryFilter) (*repositories.Repository, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	repository := &postgresRepository{}
	err := repository.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return repository.Map(), nil
}

func (r *RepositoryRepository) Single(ctx context.Context, filter *repositories.RepositoryFilter) (*repositories.Repository, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiRepositoryNotFound
	}
	return result, nil
}

func (r *RepositoryRepository) List(ctx context.Context, filter *repositories.RepositoryFilter) ([]*repositories.Repository, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var repos []*repositories.Repository
	var totalCount int
	for rows.Next() {
		repository := &postgresRepository{}
		err := repository.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		repos = append(repos, repository.Map())
	}

	return repos, totalCount, nil
}

func (r *RepositoryRepository) Insert(repository *repositories.Repository) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, repository))
}

func (r *RepositoryRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, repository *repositories.Repository) error {
	mapped := mapRepository(repository)

	s := sqlbuilder.InsertInto("repositories").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"project_id",
			"slug",
			"display_name",
			"description",
			"readme_file_id",
			"is_public",
		).
		Values(
			mapped.id,
			mapped.createdAt,
			mapped.updatedAt,
			mapped.projectId,
			mapped.slug,
			mapped.displayName,
			mapped.description,
			mapped.readmeFileId,
			mapped.isPublic,
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting repository: %w", err)
	}

	repository.SetVersion(xmin)
	repository.ClearChanges()
	return nil
}

func (r *RepositoryRepository) Update(repository *repositories.Repository) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, repository))
}

func (r *RepositoryRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, repository *repositories.Repository) error {
	if !repository.HasChanges() {
		return nil
	}

	mapped := mapRepository(repository)

	s := sqlbuilder.Update("repositories")
	s.Where(s.Equal("id", repository.GetId()))
	s.Where(s.Equal("xmin", repository.GetVersion()))

	for _, field := range repository.GetChanges() {
		switch field {
		case repositories.RepositoryChangeDisplayName:
			s.SetMore(s.Assign("display_name", mapped.displayName))
		case repositories.RepositoryChangeDescription:
			s.SetMore(s.Assign("description", mapped.description))
		case repositories.RepositoryChangeReadmeFileId:
			s.SetMore(s.Assign("readme_file_id", mapped.readmeFileId))
		case repositories.RepositoryChangeIsPublic:
			s.SetMore(s.Assign("is_public", mapped.isPublic))

		default:
			panic(fmt.Errorf("unknown repository change: %d", field))
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
		return fmt.Errorf("updating repository: %w", apiError.ErrApiConcurrentUpdate)
	}

	if err != nil {
		return fmt.Errorf("updating repository: %w", err)
	}

	repository.SetVersion(xmin)
	repository.ClearChanges()
	return nil
}

func (r *RepositoryRepository) Delete(repository *repositories.Repository) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, repository))
}

func (r *RepositoryRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, repository *repositories.Repository) error {
	s := sqlbuilder.DeleteFrom("repositories")
	s.Where(s.Equal("id", repository.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting repository: %w", err)
	}

	return nil
}
