package adminhandlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/commands"
	"github.com/the127/dockyard/internal/handlers"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/queries"
	"github.com/the127/dockyard/internal/utils/apiError"
	"github.com/the127/dockyard/internal/utils/decoding"
	"github.com/the127/dockyard/internal/utils/validate"
)

type CreateTenantRequest struct {
	Slug        string `json:"slug" validate:"required"`
	DisplayName string `json:"displayName"`
}

func CreateTenant(w http.ResponseWriter, r *http.Request) {
	var dto CreateTenantRequest
	err := decoding.HttpBodyAsJson(w, r, &dto)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	err = validate.Validate(dto)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.CreateTenantResponse](ctx, mediator, commands.CreateTenant{
		Slug:        dto.Slug,
		DisplayName: dto.DisplayName,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type ListTenantsResponse handlers.PagedResponse[ListTenantsResponseItem]

type ListTenantsResponseItem struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
}

func ListTenants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	tenants, err := mediatr.Send[*queries.ListTenantsResponse](ctx, mediator, queries.ListTenants{})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	response := ListTenantsResponse{
		Items: make([]ListTenantsResponseItem, len(tenants.Items)),
	}

	for i, item := range tenants.Items {
		response.Items[i] = ListTenantsResponseItem{
			Slug:        item.Slug,
			DisplayName: item.DisplayName,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}

type GetTenantResponse struct {
	Id          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func GetTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	vars := mux.Vars(r)
	tenantSlug := vars["tenant"]

	tenant, err := mediatr.Send[*queries.GetTenantResponse](ctx, mediator, queries.GetTenant{
		Slug: tenantSlug,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	response := GetTenantResponse{
		Id:          tenant.Id,
		Slug:        tenant.Slug,
		DisplayName: tenant.DisplayName,
		CreatedAt:   tenant.CreatedAt,
		UpdatedAt:   tenant.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}
