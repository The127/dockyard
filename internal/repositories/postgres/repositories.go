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

type repositoryRepository struct {
	tx            *sql.Tx
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresRepositoryRepository(tx *sql.Tx, changeTracker *change.Tracker, entityType int) *repositoryRepository {
	return &repositoryRepository{
		tx:            tx,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *repositoryRepository) selectQuery(filter *repositories.RepositoryFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"repositories.xmin",
		"repositories.id",
		"repositories.created_at",
		"repositories.updated_at",
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

func (r *repositoryRepository) First(ctx context.Context, filter *repositories.RepositoryFilter) (*repositories.Repository, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.tx.QueryRowContext(ctx, query, args...)

	var repository postgresRepository
	err := row.Scan(&repository.xmin, &repository.id, &repository.createdAt, &repository.updatedAt, &repository.projectId, &repository.slug, &repository.displayName, &repository.description, &repository.readmeFileId, &repository.isPublic)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return repository.Map(), nil
}

func (r *repositoryRepository) Single(ctx context.Context, filter *repositories.RepositoryFilter) (*repositories.Repository, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiRepositoryNotFound
	}
	return result, nil
}

func (r *repositoryRepository) List(ctx context.Context, filter *repositories.RepositoryFilter) ([]*repositories.Repository, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var repos []*repositories.Repository
	var totalCount int
	for rows.Next() {
		var repository postgresRepository
		err := rows.Scan(&repository.xmin, &repository.id, &repository.createdAt, &repository.updatedAt, &repository.projectId, &repository.slug, &repository.displayName, &repository.description, &repository.readmeFileId, &repository.isPublic, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		repos = append(repos, repository.Map())
	}

	return repos, totalCount, nil
}

func (r *repositoryRepository) Insert(ctx context.Context, repository *repositories.Repository) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, repository))
	return nil
}

func (r *repositoryRepository) ExecuteInsert(ctx context.Context, repository *repositories.Repository) error {
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
			repository.GetId(),
			repository.GetCreatedAt(),
			repository.GetUpdatedAt(),
			repository.GetProjectId(),
			repository.GetSlug(),
			repository.GetDisplayName(),
			repository.GetDescription(),
			repository.GetReadmeFileId(),
			repository.GetIsPublic(),
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting repository: %w", err)
	}

	repository.SetVersion(xmin)
	repository.ClearChanges()
	return nil
}

func (r *repositoryRepository) Update(ctx context.Context, repository *repositories.Repository) error {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, repository))
	return nil
}

func (r *repositoryRepository) ExecuteUpdate(ctx context.Context, repository *repositories.Repository) error {
	if !repository.HasChanges() {
		return nil
	}

	s := sqlbuilder.Update("repositories")
	s.Where(s.Equal("id", repository.GetId()))
	s.Where(s.Equal("xmin", repository.GetVersion()))

	for _, field := range repository.GetChanges() {
		switch field {
		case repositories.RepositoryChangeDisplayName:
			s.SetMore(s.Assign("display_name", repository.GetDisplayName()))
		case repositories.RepositoryChangeDescription:
			s.SetMore(s.Assign("description", repository.GetDescription()))
		case repositories.RepositoryChangeReadmeFileId:
			s.SetMore(s.Assign("readme_file_id", repository.GetReadmeFileId()))
		case repositories.RepositoryChangeIsPublic:
			s.SetMore(s.Assign("is_public", repository.GetIsPublic()))

		default:
			panic(fmt.Errorf("unknown repository change: %d", field))
		}
	}

	s.Returning("xmin")
	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.tx.QueryRowContext(ctx, query, args...)

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

func (r *repositoryRepository) Delete(ctx context.Context, repository *repositories.Repository) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, repository))
	return nil
}

func (r *repositoryRepository) ExecuteDelete(ctx context.Context, repository *repositories.Repository) error {
	s := sqlbuilder.DeleteFrom("repositories")
	s.Where(s.Equal("id", repository.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting repository: %w", err)
	}

	return nil
}
