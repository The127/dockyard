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

type postgresRepositoryBlob struct {
	postgresBaseModel
	repositoryId uuid.UUID
	blobId       uuid.UUID
}

func mapRepositoryBlob(rb *repositories.RepositoryBlob) *postgresRepositoryBlob {
	return &postgresRepositoryBlob{
		postgresBaseModel: mapBase(rb.BaseModel),
		repositoryId:      rb.GetRepositoryId(),
		blobId:            rb.GetBlobId(),
	}
}

func (rb *postgresRepositoryBlob) Map() *repositories.RepositoryBlob {
	return repositories.NewRepositoryBlobFromDB(
		rb.repositoryId,
		rb.blobId,
		rb.MapBase(),
	)
}

func (rb *postgresRepositoryBlob) scan(row RowScanner) error {
	return row.Scan(
		&rb.id,
		&rb.createdAt,
		&rb.updatedAt,
		&rb.xmin,
		&rb.repositoryId,
		&rb.blobId,
	)
}

type RepositoryBlobRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresRepositoryBlobRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *RepositoryBlobRepository {
	return &RepositoryBlobRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *RepositoryBlobRepository) selectQuery(filter *repositories.RepositoryBlobFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"repository_blobs.id",
		"repository_blobs.created_at",
		"repository_blobs.updated_at",
		"repository_blobs.xmin",
		"repository_blobs.repository_id",
		"repository_blobs.blob_id",
	).From("repository_blobs")

	if filter.HasId() {
		s.Where(s.Equal("project_accesses.id", filter.GetId()))
	}

	if filter.HasRepositoryId() {
		s.Where(s.Equal("repository_blobs.repository_id", filter.GetRepositoryId()))
	}

	if filter.HasBlobId() {
		s.Where(s.Equal("repository_blobs.blob_id", filter.GetBlobId()))
	}

	return s
}

func (r *RepositoryBlobRepository) First(ctx context.Context, filter *repositories.RepositoryBlobFilter) (*repositories.RepositoryBlob, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	repositoryBlob := &postgresRepositoryBlob{}
	err := repositoryBlob.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return repositoryBlob.Map(), nil
}

func (r *RepositoryBlobRepository) Single(ctx context.Context, filter *repositories.RepositoryBlobFilter) (*repositories.RepositoryBlob, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiRepositoryBlobNotFound
	}
	return result, nil
}

func (r *RepositoryBlobRepository) List(ctx context.Context, filter *repositories.RepositoryBlobFilter) ([]*repositories.RepositoryBlob, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var repositoryBlobs []*repositories.RepositoryBlob
	var totalCount int
	for rows.Next() {
		repositoryBlob := &postgresRepositoryBlob{}
		err := repositoryBlob.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		repositoryBlobs = append(repositoryBlobs, repositoryBlob.Map())
	}

	return repositoryBlobs, totalCount, nil
}

func (r *RepositoryBlobRepository) Insert(repositoryBlob *repositories.RepositoryBlob) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, repositoryBlob))
}

func (r *RepositoryBlobRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, repositoryBlob *repositories.RepositoryBlob) error {
	mapped := mapRepositoryBlob(repositoryBlob)

	s := sqlbuilder.InsertInto("repository_blobs").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"repository_id",
			"blob_id",
		).
		Values(
			mapped.id,
			mapped.createdAt,
			mapped.updatedAt,
			mapped.repositoryId,
			mapped.blobId,
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting repository blob: %w", err)
	}

	repositoryBlob.SetVersion(xmin)
	return nil
}

func (r *RepositoryBlobRepository) Delete(repositoryBlob *repositories.RepositoryBlob) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, repositoryBlob))
}

func (r *RepositoryBlobRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, repositoryBlob *repositories.RepositoryBlob) error {
	s := sqlbuilder.DeleteFrom("repository_blobs")
	s.Where(s.Equal("id", repositoryBlob.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting repository blob: %w", err)
	}

	return nil
}
