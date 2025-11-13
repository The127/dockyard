package apihandlers

import (
	"encoding/json"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/commands"
	"github.com/the127/dockyard/internal/handlers"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/queries"
	"github.com/the127/dockyard/internal/utils/apiError"
	"github.com/the127/dockyard/internal/utils/decoding"
	"github.com/the127/dockyard/internal/utils/validate"
)

type CreateProjectRequest struct {
	Slug        string `json:"slug" validate:"required"`
	DisplayName string `json:"displayName"`
}

func CreateProject(w http.ResponseWriter, r *http.Request) {
	var dto CreateProjectRequest
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

	vars := mux.Vars(r)
	tenantSlug := vars["tenant"]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.CreateProjectResponse](ctx, mediator, commands.CreateProject{
		TenantSlug:  tenantSlug,
		Slug:        dto.Slug,
		DisplayName: dto.DisplayName,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type ListProjectResponse handlers.PagedResponse[ListProjectResponseItem]

type ListProjectResponseItem struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
}

func ListProjects(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantSlug := vars["tenant"]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	projects, err := mediatr.Send[*queries.ListProjectsResponse](ctx, mediator, queries.ListProjects{
		TenantSlug: tenantSlug,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	response := ListProjectResponse{
		Items: make([]ListProjectResponseItem, len(projects.Items)),
	}

	for i, item := range projects.Items {
		response.Items[i] = ListProjectResponseItem{
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
