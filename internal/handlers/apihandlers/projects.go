package apihandlers

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

type CreateProjectRequest struct {
	Slug        string  `json:"slug" validate:"required"`
	DisplayName *string `json:"displayName"`
	Description *string `json:"description"`
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

	var displayName = dto.Slug
	if dto.DisplayName != nil && *dto.DisplayName != "" {
		displayName = *dto.DisplayName
	}

	_, err = mediatr.Send[*commands.CreateProjectResponse](ctx, mediator, commands.CreateProject{
		TenantSlug:  tenantSlug,
		Slug:        dto.Slug,
		DisplayName: displayName,
		Description: dto.Description,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type ListProjectResponse handlers.PagedResponse[ListProjectResponseItem]

type ListProjectResponseItem struct {
	Id          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"displayName"`
	Description *string   `json:"description"`
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
			Id:          item.Id,
			Slug:        item.Slug,
			DisplayName: item.DisplayName,
			Description: item.Description,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}

type GetProjectResponse struct {
	Id          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"displayName"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func GetProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantSlug := vars["tenant"]
	projectSlug := vars["project"]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	project, err := mediatr.Send[*queries.GetProjectResponse](ctx, mediator, queries.GetProject{
		TenantSlug:  tenantSlug,
		ProjectSlug: projectSlug,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	response := GetProjectResponse{
		Id:          project.Id,
		Slug:        project.Slug,
		DisplayName: project.DisplayName,
		Description: project.Description,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}
