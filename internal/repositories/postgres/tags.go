package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
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

type tagRepository struct {
	tx *sql.Tx
}

func NewPostgresTagRepository(tx *sql.Tx) repositories.TagRepository {
	return &tagRepository{
		tx: tx,
	}
}

func (r *tagRepository) selectQuery(filter *repositories.TagFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"tags.id",
		"tags.created_at",
		"tags.updated_at",
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
		s.Join("INNER", "manifests", "manifests.id = tags.manifest_id")
		s.SelectMore("manifests.digest as manifest_digest")
	}

	return s
}

func (r *tagRepository) First(ctx context.Context, filter *repositories.TagFilter) (*repositories.Tag, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	row := r.tx.QueryRowContext(ctx, query, args...)

	var tag postgresTag
	cols := []any{&tag.id, &tag.createdAt, &tag.updatedAt, &tag.repositoryId, &tag.manifestId, &tag.name}

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

func (r *tagRepository) Single(ctx context.Context, filter *repositories.TagFilter) (*repositories.Tag, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiTagNotFound
	}
	return result, nil
}

func (r *tagRepository) List(ctx context.Context, filter *repositories.TagFilter) ([]*repositories.Tag, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.Build()
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var tags []*repositories.Tag
	var totalCount int

	var tag postgresTag
	cols := []any{&tag.id, &tag.createdAt, &tag.updatedAt, &tag.repositoryId, &tag.manifestId, &tag.name}

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

func (r *tagRepository) Insert(ctx context.Context, tag *repositories.Tag) error {
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

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}

func (r *tagRepository) Update(ctx context.Context, tag *repositories.Tag) error {
	//TODO implement me
	panic("implement me")
}

func (r *tagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("tags")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}
