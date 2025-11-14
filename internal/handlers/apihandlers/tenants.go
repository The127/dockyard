package apihandlers

import (
	"encoding/json"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/queries"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type GetTenantOidcInfoResponse struct {
	Client string `json:"client"`
	Issuer string `json:"issuer"`
}

func GetTenantOidcInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantSlug := vars["tenant"]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	oidcInfo, err := mediatr.Send[*queries.GetTenantOidcInfoResponse](ctx, mediator, queries.GetTenantOidcInfo{
		TenantSlug: tenantSlug,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	response := GetTenantOidcInfoResponse{
		Client: oidcInfo.Client,
		Issuer: oidcInfo.Issuer,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}
