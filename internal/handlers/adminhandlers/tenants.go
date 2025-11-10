package adminhandlers

import (
	"encoding/json"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/the127/dockyard/internal/commands"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/utils/apiError"
	"github.com/the127/dockyard/internal/utils/validate"
)

type CreateTenantRequest struct {
	Slug        string `json:"slug" validate:"required"`
	DisplayName string
}

func CreateTenant(w http.ResponseWriter, r *http.Request) {
	var dto CreateTenantRequest
	err := json.NewDecoder(r.Body).Decode(&dto)
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
