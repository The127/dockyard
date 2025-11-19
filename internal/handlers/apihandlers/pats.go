package apihandlers

import (
	"encoding/json"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/commands"
	"github.com/the127/dockyard/internal/handlers"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/middlewares/authentication"
	"github.com/the127/dockyard/internal/queries"
	"github.com/the127/dockyard/internal/utils/apiError"
	"github.com/the127/dockyard/internal/utils/decoding"
	"github.com/the127/dockyard/internal/utils/validate"
)

type CreatePatRequest struct {
	DisplayName string `json:"displayName"`
}

type CreatePatResponse struct {
	Token string `json:"token"`
}

func CreatePat(w http.ResponseWriter, r *http.Request) {
	var dto CreatePatRequest
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
	currentUser := authentication.GetCurrentUser(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	pat, err := mediatr.Send[*commands.CreatePatResponse](ctx, mediator, commands.CreatePat{
		UserId:      currentUser.UserId,
		DisplayName: dto.DisplayName,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	response := CreatePatResponse{
		Token: pat.Token,
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}

type ListPatsResponse handlers.PagedResponse[ListPatsResponseItem]

type ListPatsResponseItem struct {
	Id          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
}

func ListPats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	currentUser := authentication.GetCurrentUser(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	pats, err := mediatr.Send[*queries.ListPatsResponse](ctx, mediator, queries.ListPats{
		UserId: currentUser.UserId,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	response := ListPatsResponse{
		Items: make([]ListPatsResponseItem, len(pats.Items)),
	}

	for i, item := range pats.Items {
		response.Items[i] = ListPatsResponseItem{
			Id:          item.Id,
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
