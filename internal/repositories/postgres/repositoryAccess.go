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

type postgresRepositoryAccess struct {
	postgresBaseModel
	repositoryId uuid.UUID
	userId       uuid.UUID
	role         string
}

func mapRepositoryAccess(ra *repositories.RepositoryAccess) *postgresRepositoryAccess {
	return &postgresRepositoryAccess{
		postgresBaseModel: mapBase(ra.BaseModel),
		repositoryId:      ra.GetRepositoryId(),
		userId:            ra.GetUserId(),
		role:              string(ra.GetRole()),
	}
}

func (ra *postgresRepositoryAccess) Map() *repositories.RepositoryAccess {
	return repositories.NewRepositoryAccessFromDB(
		ra.repositoryId,
		ra.userId,
		repositories.RepositoryAccessRole(ra.role),
		ra.MapBase(),
	)
}

func (ra *postgresRepositoryAccess) scan(row RowScanner) error {
	return row.Scan(
		&ra.xmin,
		&ra.id,
		&ra.createdAt,
		&ra.updatedAt,
		&ra.repositoryId,
		&ra.userId,
		&ra.role,
	)
}

type RepositoryAccessRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPostgresRepositoryAccessRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *RepositoryAccessRepository {
	return &RepositoryAccessRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *RepositoryAccessRepository) selectQuery(filter *repositories.RepositoryAccessFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"repository_accesses.id",
		"repository_accesses.created_at",
		"repository_accesses.updated_at",
		"repository_accesses.xmin",
		"repository_accesses.repository_id",
		"repository_accesses.user_id",
		"repository_accesses.role",
	).From("repository_accesses")

	if filter.HasId() {
		s.Where(s.Equal("repository_accesses.id", filter.GetId()))
	}

	if filter.HasRepositoryId() {
		s.Where(s.Equal("repository_accesses.repository_id", filter.GetRepositoryId()))
	}

	return s
}

func (r *RepositoryAccessRepository) First(ctx context.Context, filter *repositories.RepositoryAccessFilter) (*repositories.RepositoryAccess, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := r.db.QueryRowContext(ctx, query, args...)

	repositoryAccess := &postgresRepositoryAccess{}
	err := repositoryAccess.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return repositoryAccess.Map(), nil
}

func (r *RepositoryAccessRepository) List(ctx context.Context, filter *repositories.RepositoryAccessFilter) ([]*repositories.RepositoryAccess, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var repositoryAccesses []*repositories.RepositoryAccess
	var totalCount int
	for rows.Next() {
		repositoryAccess := &postgresRepositoryAccess{}
		err := repositoryAccess.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		repositoryAccesses = append(repositoryAccesses, repositoryAccess.Map())
	}

	return repositoryAccesses, totalCount, nil
}

func (r *RepositoryAccessRepository) Insert(repositoryAccess *repositories.RepositoryAccess) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, repositoryAccess))
}

func (r *RepositoryAccessRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, repositoryAccess *repositories.RepositoryAccess) error {
	mapped := mapRepositoryAccess(repositoryAccess)

	s := sqlbuilder.InsertInto("repository_accesses").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"repository_id",
			"user_id",
			"role",
		).
		Values(
			mapped.id,
			mapped.createdAt,
			mapped.updatedAt,
			mapped.repositoryId,
			mapped.userId,
			mapped.role,
		)

	s.Returning("xmin")

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32

	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting repository access: %w", err)
	}

	repositoryAccess.SetVersion(xmin)
	repositoryAccess.ClearChanges()
	return nil
}

func (r *RepositoryAccessRepository) Update(repositoryAccess *repositories.RepositoryAccess) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, repositoryAccess))
}

func (r *RepositoryAccessRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, repositoryAccess *repositories.RepositoryAccess) error {
	if !repositoryAccess.HasChanges() {
		return nil
	}

	mapped := mapRepositoryAccess(repositoryAccess)

	s := sqlbuilder.Update("repository_accesses")
	s.Where(s.Equal("id", repositoryAccess.GetId()))
	s.Where(s.Equal("xmin", repositoryAccess.GetVersion()))

	for _, field := range repositoryAccess.GetChanges() {
		switch field {
		case repositories.RepositoryAccessChangeRole:
			s.SetMore(s.Assign("role", mapped.role))
		default:
			panic(fmt.Errorf("unknown repositoryAccess change: %d", field))
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
		return fmt.Errorf("updating repository access: %w", apiError.ErrApiConcurrentUpdate)
	}

	if err != nil {
		return fmt.Errorf("updating repository access: %w", err)
	}

	repositoryAccess.SetVersion(xmin)
	repositoryAccess.ClearChanges()
	return nil
}

func (r *RepositoryAccessRepository) Delete(repositoryAccess *repositories.RepositoryAccess) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, repositoryAccess))
}

func (r *RepositoryAccessRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, repositoryAccess *repositories.RepositoryAccess) error {
	s := sqlbuilder.DeleteFrom("repository_accesses")
	s.Where(s.Equal("id", repositoryAccess.GetId()))

	query, args := s.BuildWithFlavor(sqlbuilder.PostgreSQL)
	logging.Logger.Debugf("query: %s, args: %+v", query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting repository access: %w", err)
	}

	return nil
}
