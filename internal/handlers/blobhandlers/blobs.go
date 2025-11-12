package blobhandlers

import (
	"net/http"

	"github.com/The127/ioc"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/services/blobStorage"
	"github.com/the127/dockyard/internal/utils/apiError"
)

func DownloadBlob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	digest := vars["digest"]

	blobService := ioc.GetDependency[blobStorage.Service](scope)
	err := blobService.DownloadBlob(ctx, w, digest)
	if err != nil {
		apiError.HandleHttpError(w, err)
		return
	}
}
