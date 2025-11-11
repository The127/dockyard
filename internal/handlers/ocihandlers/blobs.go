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
	"github.com/the127/dockyard/internal/services/blob"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func BlobsDownload(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func BlobExists(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func BlobsUploadStart(w http.ResponseWriter, r *http.Request) {
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

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	blobService := ioc.GetDependency[blob.Service](scope)

	uploadSession, err := blobService.StartUploadSession(ctx, blob.StartUploadSessionParams{
		BlobUploadMode: uploadMode,
	})
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Location", fmt.Sprintf("TODO: %s", uploadSession.SessionId.String()))
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
	blobService := ioc.GetDependency[blob.Service](scope)

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
	blobService := ioc.GetDependency[blob.Service](scope)

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

	computedDigest, err := blobService.CompleteUpload(ctx, sessionId)
	if err != nil {
		ociError.HandleHttpError(w, err)
		return
	}

	if computedDigest != digest {
		err = ociError.NewOciError(ociError.DigestInvalid).
			WithMessage("computed digest does not match")
		ociError.HandleHttpError(w, err)
		return
	}

	// TODO: create reference to blob in database

	w.Header().Set("Location", "TODO"+digest)
	w.WriteHeader(http.StatusCreated)
}
