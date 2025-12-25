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

type postgresTag struct {
	postgresBaseModel
	repositoryId uuid.UUID
	manifestId   uuid.UUID
	name         string
	manifestInfo *tagManifestInfo
}

type tagManifestInfo struct {
	digest string
}

func (t *postgresTag) Map() *repositories.Tag {
	var manifestInfo *repositories.TagManifestInfo
	if t.manifestInfo != nil {
		manifestInfo = &repositories.TagManifestInfo{
			Digest: t.manifestInfo.digest,
		}
	}

	tag := repositories.NewTagFromDB(
		t.repositoryId,
		t.manifestId,
		t.name,
		manifestInfo,
		t.MapBase(),
	)

	return tag
}

type TagRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresTagRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *TagRepository {
	return &TagRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *TagRepository) selectQuery(filter *repositories.TagFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"tags.id",
		"tags.created_at",
		"tags.updated_at",
		"tags.xmin",
		"tags.repository_id",
		"tags.manifest_id",
		"tags.name",
	).From("tags")

	if filter.HasId() {
		s.Where(s.Equal("tags.id", filter.GetId()))
	}

	if filter.HasRepositoryId() {
		s.Where(s.Equal("tags.repository_id", filter.GetRepositoryId()))
	}

	if filter.HasRepositoryManifestId() {
		s.Where(s.Equal("tags.manifest_id", filter.GetRepositoryManifestId()))
	}

	if filter.HasName() {
		s.Where(s.Equal("tags.name", filter.GetName()))
	}

	if filter.GetIncludeManifestInfo() {
		s.JoinWithOption(sqlbuilder.InnerJoin, "manifests", "manifests.id = tags.manifest_id")
		s.SelectMore("manifests.digest as manifest_digest")
	}

	return s
}

func (r *TagRepository) First(ctx context.Context, filter *repositories.TagFilter) (*repositories.Tag, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	var tag postgresTag
	cols := []any{&tag.id, &tag.createdAt, &tag.updatedAt, &tag.xmin, &tag.repositoryId, &tag.manifestId, &tag.name}

	if filter.GetIncludeManifestInfo() {
		tag.manifestInfo = &tagManifestInfo{}
		cols = append(cols, &tag.manifestInfo.digest)
	}

	err := row.Scan(cols...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return tag.Map(), nil
}

func (r *TagRepository) Single(ctx context.Context, filter *repositories.TagFilter) (*repositories.Tag, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiTagNotFound
	}
	return result, nil
}

func (r *TagRepository) List(ctx context.Context, filter *repositories.TagFilter) ([]*repositories.Tag, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var tags []*repositories.Tag
	var totalCount int

	var tag postgresTag
	cols := []any{&tag.id, &tag.createdAt, &tag.updatedAt, &tag.xmin, &tag.repositoryId, &tag.manifestId, &tag.name}

	if filter.GetIncludeManifestInfo() {
		tag.manifestInfo = &tagManifestInfo{}
		cols = append(cols, &tag.manifestInfo.digest)
	}

	cols = append(cols, &totalCount)

	for rows.Next() {

		err := rows.Scan(cols...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		tags = append(tags, tag.Map())
	}

	return tags, totalCount, nil
}

func (r *TagRepository) Insert(tag *repositories.Tag) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, tag))
}

func (r *TagRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, tag *repositories.Tag) error {
	s := sqlbuilder.InsertInto("tags").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"repository_id",
			"manifest_id",
			"name",
		).
		Values(
			tag.GetId(),
			tag.GetCreatedAt(),
			tag.GetUpdatedAt(),
			tag.GetRepositoryId(),
			tag.GetRepositoryManifestId(),
			tag.GetName(),
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting tag: %w", err)
	}

	tag.SetVersion(xmin)
	return nil
}

func (r *TagRepository) Delete(tag *repositories.Tag) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, tag))
}

func (r *TagRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, tag *repositories.Tag) error {
	s := sqlbuilder.DeleteFrom("tags")
	s.Where(s.Equal("id", tag.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}
