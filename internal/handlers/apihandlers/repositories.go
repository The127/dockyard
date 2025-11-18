package apihandlers

import (
	"encoding/base64"
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
	"github.com/the127/dockyard/internal/utils/pointer"
	"github.com/the127/dockyard/internal/utils/validate"
)

type CreateRepositoryRequest struct {
	Slug        string  `json:"slug" validate:"required"`
	Description *string `json:"description"`
	IsPublic    bool    `json:"isPublic"`
}

func CreateRepository(w http.ResponseWriter, r *http.Request) {
	var dto CreateRepositoryRequest
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
	projectSlug := vars["project"]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.CreateRepositoryResponse](ctx, mediator, commands.CreateRepository{
		TenantSlug:  tenantSlug,
		ProjectSlug: projectSlug,
		Slug:        dto.Slug,
		Description: dto.Description,
		IsPublic:    dto.IsPublic,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type ListRepositoriesResponse handlers.PagedResponse[ListRepositoriesResponseItem]

type ListRepositoriesResponseItem struct {
	Id          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"displayName"`
	Description *string   `json:"description"`
}

func ListRepositories(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantSlug := vars["tenant"]
	projectSlug := vars["project"]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	repos, err := mediatr.Send[*queries.ListRepositoriesResponse](ctx, mediator, queries.ListRepositories{
		TenantSlug:  tenantSlug,
		ProjectSlug: projectSlug,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	response := ListRepositoriesResponse{
		Items: make([]ListRepositoriesResponseItem, len(repos.Items)),
	}

	for i, repo := range repos.Items {
		response.Items[i] = ListRepositoriesResponseItem{
			Id:          repo.Id,
			Slug:        repo.Slug,
			DisplayName: repo.DisplayName,
			Description: repo.Description,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}

type GetRepositoryResponse struct {
	Id          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"displayName"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func GetRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantSlug := vars["tenant"]
	projectSlug := vars["project"]
	repositorySlug := vars["repository"]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	repo, err := mediatr.Send[*queries.GetRepositoryResponse](ctx, mediator, queries.GetRepository{
		TenantSlug:     tenantSlug,
		ProjectSlug:    projectSlug,
		RepositorySlug: repositorySlug,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	response := GetRepositoryResponse{
		Id:          repo.Id,
		Slug:        repo.Slug,
		DisplayName: repo.DisplayName,
		Description: repo.Description,
		CreatedAt:   repo.CreatedAt,
		UpdatedAt:   repo.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}

type GetRepositoryReadmeResponse struct {
	ContentBase64 *string `json:"contentBase64"`
}

func GetRepositoryReadme(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantSlug := vars["tenant"]
	projectSlug := vars["project"]
	repositorySlug := vars["repository"]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	readme, err := mediatr.Send[*queries.GetRepositoryReadmeResponse](ctx, mediator, queries.GetRepositoryReadme{
		TenantSlug:     tenantSlug,
		ProjectSlug:    projectSlug,
		RepositorySlug: repositorySlug,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	var contentBase64 *string
	if readme.Content != nil {
		contentBase64 = pointer.To(base64.StdEncoding.EncodeToString(*readme.Content))
	}

	response := GetRepositoryReadmeResponse{
		ContentBase64: contentBase64,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}

type UpdateRepositoryReadmeRequest struct {
	ContentBase64 string `json:"contentBase64" validate:"required"`
}

func UpdateRepositoryReadme(w http.ResponseWriter, r *http.Request) {
	var dto UpdateRepositoryReadmeRequest
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
	projectSlug := vars["project"]
	repositorySlug := vars["repository"]

	content, err := base64.StdEncoding.DecodeString(dto.ContentBase64)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.UpdateRepositoryReadmeResponse](ctx, mediator, commands.UpdateRepositoryReadme{
		TenantSlug:     tenantSlug,
		ProjectSlug:    projectSlug,
		RepositorySlug: repositorySlug,
		Content:        content,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
