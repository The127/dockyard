package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/logging"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils"
	"github.com/the127/dockyard/internal/utils/apiError"

	"github.com/huandu/go-sqlbuilder"
)

type postgresManifest struct {
	postgresBaseModel

	repositoryId uuid.UUID
	blobId       uuid.UUID

	digest string
}

func (m *postgresManifest) Map() *repositories.Manifest {
	return repositories.NewManifestFromDB(
		m.repositoryId,
		m.blobId,
		m.digest,
		m.MapBase(),
	)
}

type manifestRepository struct {
	tx            *sql.Tx
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresManifestRepository(tx *sql.Tx, changeTracker *change.Tracker, entityType int) *manifestRepository {
	return &manifestRepository{
		tx:            tx,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *manifestRepository) selectQuery(filter *repositories.ManifestFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"manifests.xmin",
		"manifests.id",
		"manifests.created_at",
		"manifests.updated_at",
		"manifests.repository_id",
		"manifests.blob_id",
		"manifests.digest",
	).From("manifests")

	if filter.HasDigest() {
		s.Where(s.Equal("manifests.digest", filter.GetDigest()))
	}

	if filter.HasId() {
		s.Where(s.Equal("manifests.id", filter.GetId()))
	}

	if filter.HasBlobId() {
		s.Where(s.Equal("manifests.blob_id", filter.GetBlobId()))
	}

	if filter.HasRepositoryId() {
		s.Where(s.Equal("manifests.repository_id", filter.GetRepositoryId()))
	}

	return s
}

func (r *manifestRepository) First(ctx context.Context, filter *repositories.ManifestFilter) (*repositories.Manifest, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.tx.QueryRowContext(ctx, query, args...)

	var manifest postgresManifest
	err := row.Scan(&manifest.xmin, &manifest.id, &manifest.createdAt, &manifest.updatedAt, &manifest.repositoryId, &manifest.blobId, &manifest.digest)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return manifest.Map(), nil
}

func (r *manifestRepository) Single(ctx context.Context, filter *repositories.ManifestFilter) (*repositories.Manifest, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiManifestNotFound
	}
	return result, nil
}

func (r *manifestRepository) List(ctx context.Context, filter *repositories.ManifestFilter) ([]*repositories.Manifest, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var manifests []*repositories.Manifest
	var totalCount int
	for rows.Next() {
		var manifest postgresManifest
		err := rows.Scan(&manifest.xmin, &manifest.id, &manifest.createdAt, &manifest.updatedAt, &manifest.repositoryId, &manifest.blobId, &manifest.digest, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		manifests = append(manifests, manifest.Map())
	}

	return manifests, totalCount, nil
}

func (r *manifestRepository) Insert(ctx context.Context, manifest *repositories.Manifest) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, manifest))
	return nil
}

func (r *manifestRepository) ExecuteInsert(ctx context.Context, manifest *repositories.Manifest) error {
	s := sqlbuilder.InsertInto("manifests").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"repository_id",
			"blob_id",
			"digest",
		).
		Values(
			manifest.GetId(),
			manifest.GetCreatedAt(),
			manifest.GetUpdatedAt(),
			manifest.GetRepositoryId(),
			manifest.GetBlobId(),
			manifest.GetDigest(),
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting manifest: %w", err)
	}

	manifest.SetVersion(xmin)
	return nil
}

func (r *manifestRepository) Delete(ctx context.Context, manifest *repositories.Manifest) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, manifest))
	return nil
}

func (r *manifestRepository) ExecuteDelete(ctx context.Context, manifest *repositories.Manifest) error {
	s := sqlbuilder.DeleteFrom("manifest")
	s.Where(s.Equal("id", manifest.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting manifest: %w", err)
	}

	return nil
}
