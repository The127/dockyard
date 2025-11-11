package ocihandlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/jsontypes"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/services/blobStorage"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func BlobsDownload(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func BlobExists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	repoIdentifier := middlewares.GetRepoIdentifier(ctx)
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	digest := vars["digest"]

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	blob, err := tx.Blobs().First(ctx, repositories.NewBlobFilter().ByDigest(digest))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if blob == nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.BlobUnknown).
			WithMessage(fmt.Sprintf("blob '%s' does not exist", digest)).
			WithHttpCode(http.StatusNotFound))
		return
	}

	tenant, err := tx.Tenants().First(ctx, repositories.NewTenantFilter().BySlug(repoIdentifier.TenantSlug))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if tenant == nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("tenant '%s' does not exist", repoIdentifier.TenantSlug)).
			WithHttpCode(http.StatusNotFound))
		return
	}

	project, err := tx.Projects().First(ctx, repositories.NewProjectFilter().ByTenantId(tenant.GetId()).BySlug(repoIdentifier.ProjectSlug))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if project == nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("project '%s' does not exist", repoIdentifier.ProjectSlug)).
			WithHttpCode(http.StatusNotFound))
		return
	}

	repository, err := tx.Repositories().First(ctx, repositories.NewRepositoryFilter().ByProjectId(project.GetId()).BySlug(repoIdentifier.RepositorySlug))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if repository == nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.NameUnknown).
			WithMessage(fmt.Sprintf("repository '%s' does not exist", repoIdentifier.RepositorySlug)).
			WithHttpCode(http.StatusNotFound))
		return
	}

	repositoryBlob, err := tx.RepositoryBlobs().First(ctx, repositories.NewRepositoryBlobFilter().ByBlobId(blob.GetId()).ByRepositoryId(repository.GetId()))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if repositoryBlob == nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.BlobUnknown).
			WithMessage(fmt.Sprintf("blob '%s' does not exist", digest)).
			WithHttpCode(http.StatusNotFound))
		return
	}

	w.Header().Set("Docker-Content-Digest", blob.GetDigest())
	w.Header().Set("Content-Length", strconv.FormatInt(blob.GetSize(), 10))
	w.WriteHeader(http.StatusOK)
}

func BlobsUploadStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	repoIdentifier := middlewares.GetRepoIdentifier(ctx)
	scope := middlewares.GetScope(ctx)

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

	digest := r.URL.Query().Get("digest")
	if digest != "" {
		err := ociError.NewOciError(ociError.Unsupported).
			WithMessage("single post upload is not supported")
		ociError.HandleHttpError(w, err)
		return
	} // check if it is a monolithic single post upload

	var uploadMode jsontypes.BlobUploadMode
	contentLength := r.Header.Get("Content-Length")
	switch contentLength {
	case "0":
		uploadMode = jsontypes.BlobUploadModeChunked

	case "":
		uploadMode = jsontypes.BlobUploadModeMonolithic

	default:
		err := ociError.NewOciError(ociError.Unsupported).
			WithMessage("unsupported content length")
		ociError.HandleHttpError(w, err)
		return
	}

	blobService := ioc.GetDependency[blobStorage.Service](scope)
	uploadSession, err := blobService.StartUploadSession(ctx, blobStorage.StartUploadSessionParams{
		BlobUploadMode: uploadMode,
		TenantSlug:     tenant.GetSlug(),
		ProjectSlug:    project.GetSlug(),
		RepositorySlug: repository.GetSlug(),
		RepositoryId:   repository.GetId(),
	})
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	var location string
	switch repoIdentifier.TenantSource {
	case middlewares.OciTenantSourcePath:
		location = fmt.Sprintf("/v2/%s/%s/%s/blobs/uploads/%s", repoIdentifier.TenantSlug, repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, uploadSession.SessionId.String())

	case middlewares.OciTenantSourceRoute:
		location = fmt.Sprintf("/v2/%s/%s/blobs/uploads/%s", repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, uploadSession.SessionId.String())

	default:
		panic(fmt.Errorf("unsupported tenant source: %s", repoIdentifier.TenantSource))
	}

	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusAccepted)
}

func UploadChunk(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/octet-stream" {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.Unsupported).
			WithMessage("unsupported content type"))
		return
	}

	lengthHeader := r.Header.Get("Content-Length")
	if lengthHeader == "" {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.Unsupported).
			WithMessage("missing content length"))
		return
	}

	length, err := strconv.Atoi(lengthHeader)
	if err != nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.Unsupported).
			WithMessage("invalid content length"))
		return
	}

	rangeHeader := r.Header.Get("Content-Range")
	if rangeHeader == "" {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.Unsupported).
			WithMessage("missing content range"))
		return
	}

	rangeParts := strings.SplitN(rangeHeader, "-", 2)

	rangeStart, err := strconv.Atoi(rangeParts[0])
	if err != nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.Unsupported).
			WithMessage("invalid content range"))
		return
	}

	rangeEnd, err := strconv.Atoi(rangeParts[1])
	if err != nil {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.Unsupported).
			WithMessage("invalid content range"))
		return
	}

	if length != rangeEnd-rangeStart+1 {
		ociError.HandleHttpError(w, ociError.NewOciError(ociError.Unsupported).
			WithMessage("content range differs from content length"))
		return
	}

	vars := mux.Vars(r)
	sessionIdString := vars["reference"]
	sessionId, err := uuid.Parse(sessionIdString)
	if err != nil {
		err := ociError.NewOciError(ociError.BlobUploadInvalid).
			WithMessage("session id must be a valid uuid")
		ociError.HandleHttpError(w, err)
		return
	}

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	blobService := ioc.GetDependency[blobStorage.Service](scope)

	err = blobService.UploadWriteChunk(ctx, sessionId, r.Body, int64(length))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Location", r.URL.String())
	w.Header().Set("Range", "0-"+strconv.Itoa(rangeEnd))
	w.WriteHeader(http.StatusAccepted)
}

func FinishUpload(w http.ResponseWriter, r *http.Request) {
	digest := r.URL.Query().Get("digest")
	if digest == "" {
		err := ociError.NewOciError(ociError.DigestInvalid).
			WithMessage("digest is required")
		ociError.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	sessionIdString := vars["reference"]
	sessionId, err := uuid.Parse(sessionIdString)
	if err != nil {
		err := ociError.NewOciError(ociError.BlobUploadInvalid).
			WithMessage("session id must be a valid uuid")
		ociError.HandleHttpError(w, err)
		return
	}

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	blobService := ioc.GetDependency[blobStorage.Service](scope)

	lengthHeader := r.Header.Get("Content-Length")
	if lengthHeader != "" {
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			ociError.HandleHttpError(w, ociError.NewOciError(ociError.Unsupported).
				WithMessage("unsupported content type"))
			return
		}

		length, err := strconv.Atoi(lengthHeader)
		if err != nil {
			ociError.HandleHttpError(w, ociError.NewOciError(ociError.Unsupported).
				WithMessage("invalid content length"))
			return
		}

		err = blobService.UploadWriteChunk(ctx, sessionId, r.Body, int64(length))
		if err != nil {
			ociError.HandleHttpError(w, err)
			return
		}
	}

	completeResponse, err := blobService.CompleteUpload(ctx, sessionId)
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	if completeResponse.ComputedDigest != digest {
		err = ociError.NewOciError(ociError.DigestInvalid).
			WithMessage("computed digest does not match")
		ociError.HandleHttpError(w, err)
		return
	}

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	blob, err := tx.Blobs().First(ctx, repositories.NewBlobFilter().ByDigest(digest))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}
	if blob == nil {
		blob = repositories.NewBlob(completeResponse.ComputedDigest, completeResponse.Size)
		err = tx.Blobs().Insert(ctx, blob)
		if err != nil {
			ociError.HandleHttpError(w, err)
			return
		}
	}

	err = tx.RepositoryBlobs().Insert(ctx, repositories.NewRepositoryBlob(completeResponse.RepositoryId, blob.GetId()))
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	repoIdentifier := middlewares.GetRepoIdentifier(ctx)

	var location string
	switch repoIdentifier.TenantSource {
	case middlewares.OciTenantSourcePath:
		location = fmt.Sprintf("/v2/%s/%s/%s/blobs/%s", repoIdentifier.TenantSlug, repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, digest)

	case middlewares.OciTenantSourceRoute:
		location = fmt.Sprintf("/v2/%s/%s/blobs/%s", repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, digest)

	default:
		panic(fmt.Errorf("unsupported tenant source: %s", repoIdentifier.TenantSource))
	}

	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}
