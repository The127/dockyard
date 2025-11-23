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

type postgresUser struct {
	postgresBaseModel
	tenantId    uuid.UUID
	subject     string
	displayName *string
	email       *string
}

func (u *postgresUser) Map() *repositories.User {
	return repositories.NewUserFromDB(
		u.tenantId,
		u.subject,
		u.displayName,
		u.email,
		u.MapBase(),
	)
}

type userRepository struct {
	tx *sql.Tx
}

func NewPostgresUserRepository(tx *sql.Tx) repositories.UserRepository {
	return &userRepository{
		tx: tx,
	}
}

func (r *userRepository) selectQuery(filter *repositories.UserFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"users.id",
		"users.created_at",
		"users.updated_at",
		"users.tenant_id",
		"users.oidc_subject",
		"users.display_name",
		"users.email",
	).From("users")

	if filter.HasId() {
		s.Where(s.Equal("users.id", filter.GetId()))
	}

	if filter.HasTenantId() {
		s.Where(s.Equal("users.tenant_id", filter.GetTenantId()))
	}

	if filter.HasSubject() {
		s.Where(s.Equal("users.subject", filter.GetSubject()))
	}

	return s
}

func (r *userRepository) First(ctx context.Context, filter *repositories.UserFilter) (*repositories.User, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	row := r.tx.QueryRowContext(ctx, query, args...)

	var user postgresUser
	err := row.Scan(&user.id, &user.createdAt, &user.updatedAt, &user.tenantId, &user.subject, &user.displayName, &user.email)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return user.Map(), nil
}

func (r *userRepository) Single(ctx context.Context, filter *repositories.UserFilter) (*repositories.User, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiUserNotFound
	}
	return result, nil
}

func (r *userRepository) List(ctx context.Context, filter *repositories.UserFilter) ([]*repositories.User, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over() as total_count")

	query, args := s.Build()
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var users []*repositories.User
	var totalCount int

	for rows.Next() {
		var user postgresUser

		err := rows.Scan(&user.id, &user.createdAt, &user.updatedAt, &user.tenantId, &user.subject, &user.displayName, &user.email, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		users = append(users, user.Map())
	}

	return users, totalCount, nil
}

func (r *userRepository) Insert(ctx context.Context, user *repositories.User) error {
	s := sqlbuilder.InsertInto("users").
		Cols(
			"id",
			"created_at",
			"updated_at",
			"tenant_id",
			"oidc_subject",
			"display_name",
			"email",
		).
		Values(
			user.GetId(),
			user.GetCreatedAt(),
			user.GetUpdatedAt(),
			user.GetTenantId(),
			user.GetSubject(),
			user.GetDisplayName(),
			user.GetEmail(),
		)

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}

func (r *userRepository) Update(ctx context.Context, user *repositories.User) error {
	//TODO implement me
	panic("implement me")
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("users")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	_, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	return nil
}
