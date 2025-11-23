package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
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

func (b *postgresBlob) Map() *repositories.Blob {
	return repositories.NewBlobFromDB(
		b.digest,
		b.size,
		b.MapBase(),
	)
}

type blobRepository struct {
	tx *sql.Tx
}

func NewPostgresBlobRepository(tx *sql.Tx) repositories.BlobRepository {
	return &blobRepository{
		tx: tx,
	}
}

func (r *blobRepository) selectQuery(filter *repositories.BlobFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"blobs.id",
		"blobs.created_at",
		"blobs.updated_at",
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

func (r *blobRepository) First(ctx context.Context, filter *repositories.BlobFilter) (*repositories.Blob, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	row := r.tx.QueryRowContext(ctx, query, args...)

	var blob postgresBlob
	err := row.Scan(&blob.id, &blob.createdAt, &blob.updatedAt, &blob.digest, &blob.size)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return blob.Map(), nil
}

func (r *blobRepository) Single(ctx context.Context, filter *repositories.BlobFilter) (*repositories.Blob, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiBlobNotFound
	}
	return result, nil
}

func (r *blobRepository) List(ctx context.Context, filter *repositories.BlobFilter) ([]*repositories.Blob, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.Build()
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var blobs []*repositories.Blob
	var totalCount int
	for rows.Next() {
		var blob postgresBlob
		err := rows.Scan(&blob.id, &blob.createdAt, &blob.updatedAt, &blob.digest, &blob.size, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		blobs = append(blobs, blob.Map())
	}

	return blobs, totalCount, nil
}

func (r *blobRepository) Insert(ctx context.Context, blob *repositories.Blob) error {
	s := sqlbuilder.InsertInto("blobs").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"digest",
			"size",
		).
		Values(
			blob.GetId(),
			blob.GetCreatedAt(),
			blob.GetUpdatedAt(),
			blob.GetDigest(),
			blob.GetSize(),
		)

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}

func (r *blobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("blob")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}
