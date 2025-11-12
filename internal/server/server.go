package server

import (
	"fmt"
	"net/http"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/handlers/adminhandlers"
	"github.com/the127/dockyard/internal/handlers/apihandlers"
	"github.com/the127/dockyard/internal/handlers/blobhandlers"
	"github.com/the127/dockyard/internal/handlers/ocihandlers"
	"github.com/the127/dockyard/internal/logging"
	"github.com/the127/dockyard/internal/middlewares"

	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func Serve(root *ioc.DependencyProvider, serverConfig config.ServerConfig) {
	r := mux.NewRouter()

	r.Use(middlewares.RecoverMiddleware())
	r.Use(middlewares.LoggingMiddleware())
	r.Use(middlewares.ScopeMiddleware(root))

	r.Use(gh.CORS(
		gh.AllowedOrigins(serverConfig.AllowedOrigins),
		gh.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "PATCH"}),
		gh.AllowedHeaders([]string{"Authorization", "Content-Type"}),
		gh.AllowCredentials(),
		gh.MaxAge(3600),
	))

	mapAdminApi(r)
	mapApi(r)
	mapOciApi(r)
	mapBlobApi(r)

	addr := fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port)
	logging.Logger.Infof("Starting server on %s", addr)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go serve(srv)
}

func serve(srv *http.Server) {
	err := srv.ListenAndServe()
	if err != nil {
		panic(fmt.Errorf("error while running server: %w", err))
	}
}

func mapBlobApi(r *mux.Router) {
	apiRouter := r.PathPrefix("/blobs/api/v1").Subrouter()
	apiRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	apiRouter.HandleFunc("/{digest}", blobhandlers.DownloadBlob).Methods(http.MethodGet, http.MethodOptions)
}

func mapAdminApi(r *mux.Router) {
	apiRouter := r.PathPrefix("/admin/api/v1").Subrouter()
	apiRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	apiRouter.HandleFunc("/tenants", adminhandlers.CreateTenant).Methods(http.MethodPost, http.MethodOptions)
}

func mapApi(r *mux.Router) {
	apiRouter := r.PathPrefix("/api/v1/tenants/{tenant}").Subrouter()
	apiRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	apiRouter.HandleFunc("/projects", apihandlers.CreateProject).Methods(http.MethodPost, http.MethodOptions)

	apiRouter.HandleFunc("/projects/{project}/repositories", apihandlers.CreateRepository).Methods(http.MethodPost, http.MethodOptions)
}

func mapOciApi(r *mux.Router) {
	apiRouter := r.PathPrefix("/v2").Subrouter()

	// implement end-1 api endpoint that shows the support for the oci api specification
	apiRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tenantProjectRepoRouter := apiRouter.PathPrefix("/{tenant}/{project}/{repository}").Subrouter()
	tenantProjectRepoRouter.Use(middlewares.OciNameMiddleware(middlewares.OciTenantSourcePath))
	mapNamedOciApi(tenantProjectRepoRouter)

	projectRepoRouter := apiRouter.PathPrefix("/{project}/{repository}").Subrouter()
	projectRepoRouter.Use(middlewares.OciNameMiddleware(middlewares.OciTenantSourceRoute))
	mapNamedOciApi(projectRepoRouter)
}

func mapNamedOciApi(r *mux.Router) {
	r.HandleFunc("/blobs/{digest}", ocihandlers.BlobsDownload).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/blobs/{digest}", ocihandlers.BlobExists).Methods(http.MethodHead, http.MethodOptions)

	r.HandleFunc("/manifests/{reference}", ocihandlers.ManifestsDownload).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/manifests/{reference}", ocihandlers.ManifestsExists).Methods(http.MethodHead, http.MethodOptions)

	r.HandleFunc("/blobs/uploads/", ocihandlers.BlobsUploadStart).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/blobs/uploads/{reference}", ocihandlers.UploadChunk).Methods(http.MethodPatch, http.MethodOptions)
	r.HandleFunc("/blobs/uploads/{reference}", ocihandlers.FinishUpload).Methods(http.MethodPut, http.MethodOptions)

	r.HandleFunc("/manifests/{reference}", ocihandlers.UploadManifest).Methods(http.MethodPut, http.MethodOptions)
}
