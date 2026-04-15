package ocihandlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/commands"
	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/middlewares/ociAuthentication"
	"github.com/the127/dockyard/internal/queries"
	"github.com/the127/dockyard/internal/services/blobStorage"
	"github.com/the127/dockyard/internal/utils/ociError"
)

var validManifestMediaTypes = map[string]bool{
	"application/vnd.oci.image.manifest.v1+json":               true,
	"application/vnd.oci.image.index.v1+json":                  true,
	"application/vnd.docker.distribution.manifest.v2+json":     true,
	"application/vnd.docker.distribution.manifest.list.v2+json": true,
}

func ManifestsDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	repoIdentifier := middlewares.GetRepoIdentifier(ctx)

	err := checkAccess(ctx, repoIdentifier, ociAuthentication.PullAccess)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	reference := vars["reference"]

	dbFactory := ioc.GetDependency[database.Factory](scope)
	dbContext, err := dbFactory.NewDbContext(ctx)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, _, repository, err := getRepositoryByIdentifier(ctx, dbContext, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	med := middlewares.GetMediator(ctx)
	result, err := mediatr.Send[*queries.GetManifestByReferenceResponse](ctx, med, queries.GetManifestByReference{
		RepositoryId: repository.GetId(),
		Reference:    reference,
	})
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	blobService := ioc.GetDependency[blobStorage.Service](scope)

	w.Header().Set("Content-Type", result.Manifest.GetMediaType())
	w.Header().Set("Docker-Content-Digest", result.Blob.GetDigest())
	w.Header().Set("Content-Length", strconv.FormatInt(result.Blob.GetSize(), 10))

	if err := blobService.DownloadBlob(ctx, w, result.Blob.GetDigest()); err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}
}

func ManifestsExists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	repoIdentifier := middlewares.GetRepoIdentifier(ctx)

	err := checkAccess(ctx, repoIdentifier, ociAuthentication.PullAccess)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	reference := vars["reference"]

	dbFactory := ioc.GetDependency[database.Factory](scope)
	dbContext, err := dbFactory.NewDbContext(ctx)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, _, repository, err := getRepositoryByIdentifier(ctx, dbContext, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	med := middlewares.GetMediator(ctx)
	result, err := mediatr.Send[*queries.GetManifestByReferenceResponse](ctx, med, queries.GetManifestByReference{
		RepositoryId: repository.GetId(),
		Reference:    reference,
	})
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", result.Manifest.GetMediaType())
	w.Header().Set("Docker-Content-Digest", result.Blob.GetDigest())
	w.Header().Set("Content-Length", strconv.FormatInt(result.Blob.GetSize(), 10))
	w.WriteHeader(http.StatusOK)
}

func UploadManifest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	repoIdentifier := middlewares.GetRepoIdentifier(ctx)

	err := checkAccess(ctx, repoIdentifier, ociAuthentication.PushAccess)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	scope := middlewares.GetScope(ctx)

	dbFactory := ioc.GetDependency[database.Factory](scope)
	dbContext, err := dbFactory.NewDbContext(ctx)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, _, repository, err := getRepositoryByIdentifier(ctx, dbContext, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*4) // max 4 MB
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		err := ociError.NewOciError(ociError.BlobUploadInvalid).
			WithMessage("failed to read manifest")
		ociError.HandleHttpError(w, r, err)
		return
	}

	var manifest struct {
		MediaType     string `json:"mediaType"`
		SchemaVersion int    `json:"schemaVersion"`
	}
	if jsonErr := json.Unmarshal(bodyBytes, &manifest); jsonErr != nil {
		ociError.HandleHttpError(w, r, ociError.NewOciError(ociError.ManifestInvalid).WithMessage("manifest is not valid JSON"))
		return
	}
	mediaType := manifest.MediaType
	if mediaType == "" {
		mediaType = r.Header.Get("Content-Type")
	}
	if !validManifestMediaTypes[mediaType] {
		ociError.HandleHttpError(w, r, ociError.NewOciError(ociError.ManifestInvalid).WithMessage("manifest mediaType is unsupported"))
		return
	}
	if manifest.SchemaVersion != 2 {
		ociError.HandleHttpError(w, r, ociError.NewOciError(ociError.ManifestInvalid).WithMessage("manifest schemaVersion must be 2"))
		return
	}

	sum256 := sha256.Sum256(bodyBytes)
	digest := "sha256:" + fmt.Sprintf("%x", sum256)

	vars := mux.Vars(r)
	reference := vars["reference"]

	med := middlewares.GetMediator(ctx)
	result, err := mediatr.Send[*commands.UploadManifestResponse](ctx, med, commands.UploadManifest{
		RepositoryId: repository.GetId(),
		Reference:    reference,
		Digest:       digest,
		MediaType:    mediaType,
		Body:         bodyBytes,
	})
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	location := fmt.Sprintf("/v2/%s/%s/manifests/%s", repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, result.Digest)

	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", result.Digest)
	w.WriteHeader(http.StatusCreated)
}
