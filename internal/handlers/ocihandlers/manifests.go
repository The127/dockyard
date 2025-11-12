package ocihandlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/services/blobStorage"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func getManifestByReference(ctx context.Context, tx database.Transaction, repositoryId uuid.UUID, reference string) (*repositories.Manifest, *repositories.Blob, error) {
	if !strings.HasPrefix(reference, "sha256:") {
		tag, err
	}

	manifest, err := tx.Manifests().First(ctx, repositories.NewManifestFilter().ByRepositoryId(repositoryId).ByDigest(reference))
	if err != nil {
		return nil, nil, err
	}
	if manifest == nil {
		err := ociError.NewOciError(ociError.ManifestUnknown).
			WithMessage(fmt.Sprintf("manifest '%s' does not exist", reference))
		return nil, nil, err
	}

	blob, err := tx.Blobs().First(ctx, repositories.NewBlobFilter().ById(manifest.GetBlobId()))
	if err != nil {
		return nil, nil, err
	}
	if blob == nil {
		err := ociError.NewOciError(ociError.BlobUnknown).
			WithMessage(fmt.Sprintf("blob '%s' does not exist", manifest.GetBlobId()))
		return nil, nil, err
	}

	return manifest, blob, nil
}

func ManifestsDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	repoIdentifier := middlewares.GetRepoIdentifier(ctx)

	vars := mux.Vars(r)
	reference := vars["reference"]

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	repository, err := getRepositoryByIdentifier(ctx, tx, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	_, blob, err := getManifestByReference(ctx, tx, repository.GetId(), reference)
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	blobService := ioc.GetDependency[blobStorage.Service](scope)
	redirectUri, err := blobService.GetBlobDownloadLink(ctx, blob.GetDigest())
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	http.Redirect(w, r, redirectUri, http.StatusTemporaryRedirect)
}

func ManifestsExists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	repoIdentifier := middlewares.GetRepoIdentifier(ctx)
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	reference := vars["reference"]

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	repository, err := getRepositoryByIdentifier(ctx, tx, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	_, blob, err := getManifestByReference(ctx, tx, repository.GetId(), reference)
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Docker-Content-Digest", blob.GetDigest())
	w.Header().Set("Content-Length", strconv.FormatInt(blob.GetSize(), 10))
	w.WriteHeader(http.StatusOK)
}

func UploadManifest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	repoIdentifier := middlewares.GetRepoIdentifier(ctx)

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	repository, err := getRepositoryByIdentifier(ctx, tx, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*1024*4) // max 4 MB
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.BlobUploadInvalid).
			WithMessage("failed to read manifest"))
		return
	}

	// TODO: validate manifest

	blobs := ioc.GetDependency[blobStorage.Service](scope)
	uploadResponse, err := blobs.UploadCompleteBlob(ctx, bytes.NewReader(bodyBytes))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	blob, err := tx.Blobs().First(ctx, repositories.NewBlobFilter().ByDigest(uploadResponse.Digest))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if blob == nil {
		blob = repositories.NewBlob(uploadResponse.Digest, int64(len(bodyBytes)))
		if err := tx.Blobs().Insert(ctx, blob); err != nil {
			ociError.HandleHttpError(w, err)
			return
		}
	}

	err = tx.Manifests().Insert(ctx, repositories.NewManifest(repository.GetId(), blob.GetId(), uploadResponse.Digest))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	var location string
	switch repoIdentifier.TenantSource {
	case middlewares.OciTenantSourceRoute:
		location = fmt.Sprintf("/v2/%s/%s/manifests/%s", repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, uploadResponse.Digest)

	case middlewares.OciTenantSourcePath:
		location = fmt.Sprintf("/v2/%s/%s/%s/manifests/%s", repoIdentifier.TenantSlug, repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, uploadResponse.Digest)

	default:
		panic(fmt.Errorf("unsupported tenant source: %s", repoIdentifier.TenantSource))
	}

	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}
