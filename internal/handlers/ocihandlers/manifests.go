package ocihandlers

import (
	"crypto/sha256"
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

	med := ioc.GetDependency[mediatr.Mediator](scope)
	result, err := mediatr.Send[*queries.GetManifestByReferenceResponse](ctx, med, queries.GetManifestByReference{
		RepositoryId: repository.GetId(),
		Reference:    reference,
	})
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	blobService := ioc.GetDependency[blobStorage.Service](scope)
	redirectUri, err := blobService.GetBlobDownloadLink(ctx, result.Blob.GetDigest())
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	http.Redirect(w, r, redirectUri, http.StatusTemporaryRedirect)
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

	med := ioc.GetDependency[mediatr.Mediator](scope)
	result, err := mediatr.Send[*queries.GetManifestByReferenceResponse](ctx, med, queries.GetManifestByReference{
		RepositoryId: repository.GetId(),
		Reference:    reference,
	})
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

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

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*1024*4) // max 4 MB
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		err := ociError.NewOciError(ociError.BlobUploadInvalid).
			WithMessage("failed to read manifest")
		ociError.HandleHttpError(w, r, err)
		return
	}

	// TODO: validate manifest

	sum256 := sha256.Sum256(bodyBytes)
	digest := "sha256:" + fmt.Sprintf("%x", sum256)

	vars := mux.Vars(r)
	reference := vars["reference"]

	med := ioc.GetDependency[mediatr.Mediator](scope)
	result, err := mediatr.Send[*commands.UploadManifestResponse](ctx, med, commands.UploadManifest{
		RepositoryId: repository.GetId(),
		Reference:    reference,
		Digest:       digest,
		Body:         bodyBytes,
	})
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	location := fmt.Sprintf("/v2/%s/%s/manifests/%s", repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, result.Digest)

	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}
