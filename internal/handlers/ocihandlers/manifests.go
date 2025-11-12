package ocihandlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/The127/ioc"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/services/blobStorage"
	"github.com/the127/dockyard/internal/utils/ociError"
)

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

	manifest, err := tx.Manifests().First(ctx, repositories.NewManifestFilter().ByRepositoryId(repository.GetId()).ByReference(reference))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if manifest == nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.ManifestUnknown).
			WithMessage(fmt.Sprintf("manifest '%s' does not exist", reference)))
		return
	}

	blob, err := tx.Blobs().First(ctx, repositories.NewBlobFilter().ById(manifest.GetBlobId()))
	if err != nil {
		ociError.HandleHttpError(w, err)
	}
	if blob == nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.BlobUnknown).
			WithMessage(fmt.Sprintf("blob '%s' does not exist", manifest.GetBlobId())))
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

	tenant, err := tx.Tenants().First(ctx, repositories.NewTenantFilter().BySlug(repoIdentifier.TenantSlug))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if tenant == nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("tenant '%s' does not exist", repoIdentifier.TenantSlug)))
		return
	}

	project, err := tx.Projects().First(ctx, repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(repoIdentifier.ProjectSlug))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if project == nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("project '%s' does not exist", repoIdentifier.ProjectSlug)))
		return
	}

	repository, err := tx.Repositories().First(ctx, repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(repoIdentifier.RepositorySlug))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if repository == nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("repository '%s' does not exist", repoIdentifier.RepositorySlug)))
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
