package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/logging"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils"
	"github.com/the127/dockyard/internal/utils/apiError"

	"github.com/huandu/go-sqlbuilder"
)

type postgresBlob struct {
	postgresBaseModel
	digest string
	size   int64
}

func mapBlob(blob *repositories.Blob) *postgresBlob {
	return &postgresBlob{
		postgresBaseModel: mapBase(blob.BaseModel),
		digest:            blob.GetDigest(),
		size:              blob.GetSize(),
	}
}

func (b *postgresBlob) Map() *repositories.Blob {
	return repositories.NewBlobFromDB(
		b.digest,
		b.size,
		b.MapBase(),
	)
}

func (b *postgresBlob) scan(row RowScanner) error {
	return row.Scan(
		&b.id,
		&b.createdAt,
		&b.updatedAt,
		&b.xmin,
		&b.digest,
		&b.size,
	)
}

type BlobRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresBlobRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *BlobRepository {
	return &BlobRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *BlobRepository) selectQuery(filter *repositories.BlobFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"blobs.id",
		"blobs.created_at",
		"blobs.updated_at",
		"blobs.xmin",
		"blobs.digest",
		"blobs.size",
	).From("blobs")

	if filter.HasDigest() {
		s.Where(s.Equal("blobs.digest", filter.GetDigest()))
	}

	if filter.HasId() {
		s.Where(s.Equal("blobs.id", filter.GetId()))
	}

	return s
}

func (r *BlobRepository) First(ctx context.Context, filter *repositories.BlobFilter) (*repositories.Blob, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	blob := &postgresBlob{}
	err := blob.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return blob.Map(), nil
}

func (r *BlobRepository) Single(ctx context.Context, filter *repositories.BlobFilter) (*repositories.Blob, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiBlobNotFound
	}
	return result, nil
}

func (r *BlobRepository) List(ctx context.Context, filter *repositories.BlobFilter) ([]*repositories.Blob, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var blobs []*repositories.Blob
	var totalCount int
	for rows.Next() {
		blob := &postgresBlob{}
		err := blob.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		blobs = append(blobs, blob.Map())
	}

	return blobs, totalCount, nil
}

func (r *BlobRepository) Insert(blob *repositories.Blob) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, blob))
}

func (r *BlobRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, blob *repositories.Blob) error {
	mapped := mapBlob(blob)

	s := sqlbuilder.InsertInto("blobs").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"digest",
			"size",
		).
		Values(
			mapped.id,
			mapped.createdAt,
			mapped.updatedAt,
			mapped.digest,
			mapped.size,
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting blob: %w", err)
	}

	blob.SetVersion(xmin)
	return nil
}

func (r *BlobRepository) Delete(blob *repositories.Blob) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, blob))
}

func (r *BlobRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, blob *repositories.Blob) error {
	s := sqlbuilder.DeleteFrom("blob")
	s.Where(s.Equal("id", blob.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting blob: %w", err)
	}

	return nil
}
