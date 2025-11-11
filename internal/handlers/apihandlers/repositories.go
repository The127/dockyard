package apihandlers

import (
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/commands"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/utils/apiError"
	"github.com/the127/dockyard/internal/utils/decoding"
	"github.com/the127/dockyard/internal/utils/validate"
)

type CreateRepositoryRequest struct {
	Slug        string `json:"slug" validate:"required"`
	DisplayName string `json:"displayName"`
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
		DisplayName: dto.DisplayName,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
