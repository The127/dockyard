package ocihandlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/jsontypes"
	"github.com/the127/dockyard/internal/middlewares"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/services/blobStorage"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func getRepositoryBlob(ctx context.Context, tx database.Transaction, repositoryId uuid.UUID, digest string) (*repositories.RepositoryBlob, *repositories.Blob, error) {
	blob, err := tx.Blobs().First(ctx, repositories.NewBlobFilter().ByDigest(digest))
	if err != nil {
		return nil, nil, err
	}
	if blob == nil {
		err := ociError.NewOciError(ociError.BlobUnknown).
			WithMessage(fmt.Sprintf("blob '%s' does not exist", digest)).
			WithHttpCode(http.StatusNotFound)
		return nil, nil, err
	}

	repositoryBlob, err := tx.RepositoryBlobs().First(ctx, repositories.NewRepositoryBlobFilter().ByBlobId(blob.GetId()).ByRepositoryId(repositoryId))
	if err != nil {
		return nil, nil, err
	}
	if repositoryBlob == nil {
		err := ociError.NewOciError(ociError.BlobUnknown).
			WithMessage(fmt.Sprintf("blob '%s' does not exist", digest)).
			WithHttpCode(http.StatusNotFound)
		return nil, nil, err
	}

	return repositoryBlob, blob, nil
}

func BlobsDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	repoIdentifier := middlewares.GetRepoIdentifier(ctx)
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	digest := vars["digest"]

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, _, repository, err := getRepositoryByIdentifier(ctx, tx, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
	}

	err = checkAccess(ctx, tx, repoIdentifier, repository, "pull")
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, blob, err := getRepositoryBlob(ctx, tx, repository.GetId(), digest)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
	}

	blobService := ioc.GetDependency[blobStorage.Service](scope)
	redirectUri, err := blobService.GetBlobDownloadLink(ctx, blob.GetDigest())
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	http.Redirect(w, r, redirectUri, http.StatusTemporaryRedirect)
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
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, _, repository, err := getRepositoryByIdentifier(ctx, tx, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
	}

	err = checkAccess(ctx, tx, repoIdentifier, repository, "pull")
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, blob, err := getRepositoryBlob(ctx, tx, repository.GetId(), digest)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
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
		ociError.HandleHttpError(w, r, err)
		return
	}

	tenant, project, repository, err := getRepositoryByIdentifier(ctx, tx, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
	}

	err = checkAccess(ctx, tx, repoIdentifier, repository, "push")
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	digest := r.URL.Query().Get("digest")
	if digest != "" {
		err := ociError.NewOciError(ociError.Unsupported).
			WithMessage("single post upload is not supported")
		ociError.HandleHttpError(w, r, err)
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
		ociError.HandleHttpError(w, r, err)
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
		ociError.HandleHttpError(w, r, err)
		return
	}

	location := fmt.Sprintf("/v2/%s/%s/blobs/uploads/%s", repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, uploadSession.SessionId.String())

	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusAccepted)
}

func UploadChunk(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/octet-stream" {
		err := ociError.NewOciError(ociError.Unsupported).
			WithMessage("unsupported content type")
		ociError.HandleHttpError(w, r, err)
		return
	}

	/*lengthHeader := r.Header.Get("Content-Length")
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
	}*/

	vars := mux.Vars(r)
	sessionIdString := vars["reference"]
	sessionId, err := uuid.Parse(sessionIdString)
	if err != nil {
		err := ociError.NewOciError(ociError.BlobUploadInvalid).
			WithMessage("session id must be a valid uuid")
		ociError.HandleHttpError(w, r, err)
		return
	}

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	blobService := ioc.GetDependency[blobStorage.Service](scope)

	uploadResponse, err := blobService.UploadWriteChunk(ctx, sessionId, r.Body)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	w.Header().Set("Location", r.URL.String())
	w.Header().Set("Range", "0-"+strconv.FormatInt(uploadResponse.Size, 10))
	w.WriteHeader(http.StatusAccepted)
}

func FinishUpload(w http.ResponseWriter, r *http.Request) {
	digest := r.URL.Query().Get("digest")
	if digest == "" {
		err := ociError.NewOciError(ociError.DigestInvalid).
			WithMessage("digest is required")
		ociError.HandleHttpError(w, r, err)
		return
	}

	vars := mux.Vars(r)
	sessionIdString := vars["reference"]
	sessionId, err := uuid.Parse(sessionIdString)
	if err != nil {
		err := ociError.NewOciError(ociError.BlobUploadInvalid).
			WithMessage("session id must be a valid uuid")
		ociError.HandleHttpError(w, r, err)
		return
	}

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	blobService := ioc.GetDependency[blobStorage.Service](scope)

	/*lengthHeader := r.Header.Get("Content-Length")
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

		err := blobService.UploadWriteChunk(ctx, sessionId, r.Body)
		if err != nil {
			ociError.HandleHttpError(w, r, err)
			return
		}
	}*/

	_, err = blobService.UploadWriteChunk(ctx, sessionId, r.Body)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	completeResponse, err := blobService.CompleteUpload(ctx, sessionId, blobStorage.BlobContentTypeOctetStream)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	if completeResponse.ComputedDigest != digest {
		err = ociError.NewOciError(ociError.DigestInvalid).
			WithMessage("computed digest does not match")
		ociError.HandleHttpError(w, r, err)
		return
	}

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	blob, err := tx.Blobs().First(ctx, repositories.NewBlobFilter().ByDigest(digest))
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}
	if blob == nil {
		blob = repositories.NewBlob(completeResponse.ComputedDigest, completeResponse.Size)
		err = tx.Blobs().Insert(ctx, blob)
		if err != nil {
			ociError.HandleHttpError(w, r, err)
			return
		}
	}

	err = tx.RepositoryBlobs().Insert(ctx, repositories.NewRepositoryBlob(completeResponse.RepositoryId, blob.GetId()))
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	repoIdentifier := middlewares.GetRepoIdentifier(ctx)

	location := fmt.Sprintf("/v2/%s/%s/%s/blobs/%s", repoIdentifier.TenantSlug, repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, digest)

	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}
