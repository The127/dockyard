package inmemory

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/change"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type TenantRepository struct {
	txn           *memdb.Txn
	changeTracker *change.Tracker
	entityType    int
}

func NewInMemoryTenantRepository(txn *memdb.Txn, changeTracker *change.Tracker, entityType int) *TenantRepository {
	return &TenantRepository{
		txn:           txn,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *TenantRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.TenantFilter) ([]*repositories.Tenant, int) {
	var result []*repositories.Tenant

	obj := iterator.Next()
	for obj != nil {
		typed := obj.(repositories.Tenant)

		if r.matches(&typed, filter) {
			result = append(result, &typed)
		}

		obj = iterator.Next()
	}

	count := len(result)

	return result, count
}

func (r *TenantRepository) matches(tenant *repositories.Tenant, filter *repositories.TenantFilter) bool {
	if filter.HasSlug() {
		if tenant.GetSlug() != filter.GetSlug() {
			return false
		}
	}

	if filter.HasId() {
		if tenant.GetId() != filter.GetId() {
			return false
		}
	}

	return true
}

func (r *TenantRepository) First(_ context.Context, filter *repositories.TenantFilter) (*repositories.Tenant, error) {
	iterator, err := r.txn.Get("tenants", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get tenants: %w", err)
	}

	result, _ := r.applyFilter(iterator, filter)

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *TenantRepository) Single(_ context.Context, filter *repositories.TenantFilter) (*repositories.Tenant, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiTenantNotFound
	}
	return result, nil
}

func (r *TenantRepository) List(_ context.Context, filter *repositories.TenantFilter) ([]*repositories.Tenant, int, error) {
	iterator, err := r.txn.Get("tenants", "id")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tenants: %w", err)
	}

	result, count := r.applyFilter(iterator, filter)

	return result, count, err
}

func (r *TenantRepository) Insert(_ context.Context, tenant *repositories.Tenant) error {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, tenant))
	return nil
}

func (r *TenantRepository) ExecuteInsert(tx *memdb.Txn, tenant *repositories.Tenant) error {
	err := tx.Insert("tenants", *tenant)
	if err != nil {
		return fmt.Errorf("failed to insert tenant: %w", err)
	}

	tenant.ClearChanges()
	return nil
}

func (r *TenantRepository) Update(_ context.Context, tenant *repositories.Tenant) error {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, tenant))
	return nil
}

func (r *TenantRepository) ExecuteUpdate(tx *memdb.Txn, tenant *repositories.Tenant) error {
	err := r.txn.Insert("tenants", *tenant)
	if err != nil {
		return fmt.Errorf("failed to insert tenant: %w", err)
	}

	tenant.ClearChanges()
	return nil
}

func (r *TenantRepository) Delete(_ context.Context, tenant *repositories.Tenant) error {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, tenant))
	return nil
}

func (r *TenantRepository) ExecuteDelete(tx *memdb.Txn, tenant *repositories.Tenant) error {
	err := r.txn.Delete("tenants", tenant)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	return nil
}
