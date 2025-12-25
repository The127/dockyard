package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/huandu/go-sqlbuilder"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/logging"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type postgresFile struct {
	postgresBaseModel
	digest      string
	contentType string
	data        []byte
	size        int64
}

func (f *postgresFile) Map() *repositories.File {
	return repositories.NewFileFromDB(
		f.digest,
		f.contentType,
		f.data,
		f.size,
		f.MapBase(),
	)
}

type FileRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresFileRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *FileRepository {
	return &FileRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *FileRepository) selectQuery(filter *repositories.FileFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"files.xmin",
		"files.id",
		"files.created_at",
		"files.updated_at",
		"files.digest",
		"files.content_type",
		"files.data",
		"files.size",
	).From("files")

	if filter.HasId() {
		s.Where(s.Equal("files.id", filter.GetId()))
	}

	if filter.HasDigest() {
		s.Where(s.Equal("files.digest", filter.GetDigest()))
	}

	return s
}

func (r *FileRepository) First(ctx context.Context, filter *repositories.FileFilter) (*repositories.File, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	var file postgresFile
	err := row.Scan(&file.xmin, &file.id, &file.createdAt, &file.updatedAt, &file.digest, &file.contentType, &file.data, &file.size)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return file.Map(), nil
}

func (r *FileRepository) Single(ctx context.Context, filter *repositories.FileFilter) (*repositories.File, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiFileNotFound
	}
	return result, nil
}

func (r *FileRepository) List(ctx context.Context, filter *repositories.FileFilter) ([]*repositories.File, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var files []*repositories.File
	var totalCount int
	for rows.Next() {
		var file postgresFile
		err := rows.Scan(&file.xmin, &file.id, &file.createdAt, &file.updatedAt, &file.digest, &file.contentType, &file.data, &file.size, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		files = append(files, file.Map())
	}

	return files, totalCount, nil
}

func (r *FileRepository) Insert(ctx context.Context, file *repositories.File) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, file))
	return nil
}

func (r *FileRepository) ExecuteInsert(ctx context.Context, file *repositories.File) error {
	s := sqlbuilder.InsertInto("files").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"digest",
			"content_type",
			"data",
			"size",
		).
		Values(
			file.GetId(),
			file.GetCreatedAt(),
			file.GetUpdatedAt(),
			file.GetDigest(),
			file.GetContentType(),
			file.GetData(),
			file.GetSize(),
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting file: %w", err)
	}

	file.SetVersion(xmin)
	return nil
}

func (r *FileRepository) Delete(ctx context.Context, file *repositories.File) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, file))
	return nil
}

func (r *FileRepository) ExecuteDelete(ctx context.Context, file *repositories.File) error {
	s := sqlbuilder.DeleteFrom("files")
	s.Where(s.Equal("id", file.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}

	return nil
}
