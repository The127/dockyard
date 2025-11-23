package inmemory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type tenantRepository struct {
	txn *memdb.Txn
}

func NewInMemoryTenantRepository(txn *memdb.Txn) repositories.TenantRepository {
	return &tenantRepository{
		txn: txn,
	}
}

func (r *tenantRepository) applyFilter(iterator memdb.ResultIterator, filter *repositories.TenantFilter) ([]*repositories.Tenant, int, error) {
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

	return result, count, nil
}

func (r *tenantRepository) matches(tenant *repositories.Tenant, filter *repositories.TenantFilter) bool {
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

func (r *tenantRepository) First(_ context.Context, filter *repositories.TenantFilter) (*repositories.Tenant, error) {
	iterator, err := r.txn.Get("tenants", "id")
	if err != nil {
		return nil, fmt.Errorf("failed to get tenants: %w", err)
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

func (r *tenantRepository) Single(_ context.Context, filter *repositories.TenantFilter) (*repositories.Tenant, error) {
	result, err := r.First(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apiError.ErrApiTenantNotFound
	}
	return result, nil
}

func (r *tenantRepository) List(_ context.Context, filter *repositories.TenantFilter) ([]*repositories.Tenant, int, error) {
	iterator, err := r.txn.Get("tenants", "id")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tenants: %w", err)
	}

	result, count, err := r.applyFilter(iterator, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to apply filter: %w", err)
	}

	return result, count, err
}

func (r *tenantRepository) Insert(_ context.Context, tenant *repositories.Tenant) error {
	err := r.txn.Insert("tenants", *tenant)
	if err != nil {
		return fmt.Errorf("failed to insert tenant: %w", err)
	}

	tenant.ClearChanges()
	return nil
}

func (r *tenantRepository) Update(_ context.Context, tenant *repositories.Tenant) error {
	err := r.txn.Insert("tenants", *tenant)
	if err != nil {
		return fmt.Errorf("failed to insert tenant: %w", err)
	}

	tenant.ClearChanges()
	return nil
}

func (r *tenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	entry, err := r.First(ctx, repositories.NewTenantFilter().ById(id))
	if err != nil {
		return fmt.Errorf("failed to get by id: %w", err)
	}
	if entry == nil {
		return nil
	}

	return r.txn.Delete("tenants", entry)
}
