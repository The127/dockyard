package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type userRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryUserRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) *userRepository {
	return &userRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *userRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.UserFilter) ([]*repositories.User, int, error) {
	var result []*repositories.User

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.User)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count, nil
}

func (r *userRepository) matches(user *repositories.User, filter *repositories.UserFilter) bool {
	if filter.HasTenantId() {
		if user.GetTenantId() != filter.GetTenantId() {
			return false
		}
	}

	if filter.HasSubject() {
		if user.GetSubject() != filter.GetSubject() {
			return false
		}
	}

	if filter.HasId() {
		if user.GetId() != filter.GetId() {
			return false
		}
	}

	return true
}

func (r *userRepository) First(_ context.Context, filter *repositories.UserFilter) (*repositories.User, error) {
	iterator, err := r.txn.Get("users", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	result, _, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to apply filter: %w", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *userRepository) Single(_ context.Context, filter *repositories.UserFilter) (*repositories.User, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiUserNotFound
	}
	return result, nil
}

func (r *userRepository) List(_ context.Context, filter *repositories.UserFilter) ([]*repositories.User, int, error) {
	iterator, err := r.txn.Get("users", "id")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	return r.applyFilter(iterator, filter)
}

func (r *userRepository) Insert(_ context.Context, user *repositories.User) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, user))
	return nil
}

func (r *userRepository) ExecuteInsert(_ context.Context, user *repositories.User) error {
	err := r.txn.Insert("users", *user)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	user.ClearChanges()
	return nil
}

func (r *userRepository) Update(_ context.Context, user *repositories.User) error {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, user))
	return nil
}

func (r *userRepository) ExecuteUpdate(_ context.Context, user *repositories.User) error {
	err := r.txn.Insert("users", *user)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	user.ClearChanges()
	return nil
}

func (r *userRepository) Delete(_ context.Context, user *repositories.User) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, user))
	return nil
}

func (r *userRepository) ExecuteDelete(_ context.Context, user *repositories.User) error {
	err := r.txn.Delete("users", user)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
