package server

import (
	"encoding/json"
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
	"github.com/the127/dockyard/internal/middlewares/authentication"

	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func Serve(root *ioc.DependencyProvider, serverConfig config.ServerConfig, hostBlobApi bool) {
	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logging.Logger.Infof("Not found API Request: %s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]string{
				{"code": "NOT_FOUND", "message": "route not found"},
			},
		})
	})

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

	if hostBlobApi {
		mapBlobApi(r)
	}

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

	// TODO: implement authentication
	apiRouter.HandleFunc("/{digest}", blobhandlers.DownloadBlob).Methods(http.MethodGet, http.MethodOptions)
}

func mapAdminApi(r *mux.Router) {
	apiRouter := r.PathPrefix("/admin/api/v1").Subrouter()
	apiRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// unauthenticated endpoints need to go above the authentication middleware
	authApiRouter := apiRouter.PathPrefix("").Subrouter()
	// TODO: make it work for non {tenant} routes
	// authApiRouter.Use(authentication.ApiAuthenticationMiddleware())

	authApiRouter.HandleFunc("/tenants", adminhandlers.CreateTenant).Methods(http.MethodPost, http.MethodOptions)
	authApiRouter.HandleFunc("/tenants", adminhandlers.ListTenants).Methods(http.MethodGet, http.MethodOptions)
	authApiRouter.HandleFunc("/tenants/{tenant}", adminhandlers.GetTenant).Methods(http.MethodGet, http.MethodOptions)
}

func mapApi(r *mux.Router) {
	apiRouter := r.PathPrefix("/api/v1/tenants/{tenant}").Subrouter()
	apiRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	apiRouter.HandleFunc("/oidc", apihandlers.GetTenantOidcInfo).Methods(http.MethodGet, http.MethodOptions)

	// unauthenticated endpoints need to go above the authentication middleware
	authApiRouter := apiRouter.PathPrefix("").Subrouter()
	authApiRouter.Use(authentication.ApiAuthenticationMiddleware())

	authApiRouter.HandleFunc("/projects", apihandlers.CreateProject).Methods(http.MethodPost, http.MethodOptions)
	authApiRouter.HandleFunc("/projects", apihandlers.ListProjects).Methods(http.MethodGet, http.MethodOptions)
	authApiRouter.HandleFunc("/projects/{project}", apihandlers.GetProject).Methods(http.MethodGet, http.MethodOptions)

	authApiRouter.HandleFunc("/projects/{project}/repositories", apihandlers.CreateRepository).Methods(http.MethodPost, http.MethodOptions)
	authApiRouter.HandleFunc("/projects/{project}/repositories", apihandlers.ListRepositories).Methods(http.MethodGet, http.MethodOptions)
	authApiRouter.HandleFunc("/projects/{project}/repositories/{repository}", apihandlers.GetRepository).Methods(http.MethodGet, http.MethodOptions)
	authApiRouter.HandleFunc("/projects/{project}/repositories/{repository}", apihandlers.PatchRepository).Methods(http.MethodPatch, http.MethodOptions)

	authApiRouter.HandleFunc("/projects/{project}/repositories/{repository}/readme", apihandlers.GetRepositoryReadme).Methods(http.MethodGet, http.MethodOptions)
	authApiRouter.HandleFunc("/projects/{project}/repositories/{repository}/readme", apihandlers.UpdateRepositoryReadme).Methods(http.MethodPut, http.MethodOptions)

	authApiRouter.HandleFunc("/projects/{project}/repositories/{repository}/tags", apihandlers.ListTags).Methods(http.MethodGet, http.MethodOptions)
}

func mapOciApi(r *mux.Router) {
	apiRouter := r.PathPrefix("/v2").Subrouter()
	apiRouter.HandleFunc("/tokens", ocihandlers.Tokens).Methods(http.MethodPost, http.MethodGet, http.MethodOptions)

	// unauthenticated endpoints need to go above the authentication middleware
	authApiRouter := apiRouter.PathPrefix("").Subrouter()
	authApiRouter.Use(authentication.OciAuthenticationMiddleware())

	// implement end-1 api endpoint that shows the support for the oci api specification
	authApiRouter.HandleFunc("/", ocihandlers.Root).Methods(http.MethodGet, http.MethodOptions)

	tenantProjectRepoRouter := authApiRouter.PathPrefix("/{tenant}/{project}/{repository}").Subrouter()
	tenantProjectRepoRouter.Use(middlewares.OciNameMiddleware(middlewares.OciTenantSourcePath))
	mapNamedOciApi(tenantProjectRepoRouter)

	projectRepoRouter := authApiRouter.PathPrefix("/{project}/{repository}").Subrouter()
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
