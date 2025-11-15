package apihandlers

import (
	"encoding/json"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/handlers"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/queries"
	"github.com/the127/dockyard/internal/utils/apiError"
)

type ListTagsResponse handlers.PagedResponse[ListTagsResponseItem]

type ListTagsResponseItem struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

func ListTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantSlug := vars["tenant"]
	projectSlug := vars["project"]
	repositorySlug := vars["repository"]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	mediator := ioc.GetDependency[mediatr.Mediator](scope)

	tags, err := mediatr.Send[*queries.ListTagsResponse](ctx, mediator, queries.ListTags{
		TenantSlug:     tenantSlug,
		ProjectSlug:    projectSlug,
		RepositorySlug: repositorySlug,
	})
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}

	response := ListTagsResponse{
		Items: make([]ListTagsResponseItem, len(tags.Items)),
	}

	for i, tag := range tags.Items {
		response.Items[i] = ListTagsResponseItem{
			Name:   tag.Name,
			Digest: tag.Digest,
			Size:   tag.Size,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}
