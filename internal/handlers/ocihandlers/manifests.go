package ocihandlers

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	"github.com/the127/dockyard/internal/middlewares/ociAuthentication"
	"github.com/the127/dockyard/internal/repositories"
	"github.com/the127/dockyard/internal/services"
	"github.com/the127/dockyard/internal/services/blobStorage"
	"github.com/the127/dockyard/internal/utils/ociError"
)

func getManifestByReference(ctx context.Context, tx database.Transaction, repositoryId uuid.UUID, reference string) (*repositories.Manifest, *repositories.Blob, error) { // nolint:unparam
	var manifestFilter *repositories.ManifestFilter
	if !strings.HasPrefix(reference, "sha256:") {
		tag, err := tx.Tags().First(ctx, repositories.NewTagFilter().ByRepositoryId(repositoryId).ByName(reference))
		if err != nil {
			return nil, nil, err
		}
		if tag == nil {
			err := ociError.NewOciError(ociError.ManifestUnknown).
				WithMessage(fmt.Sprintf("tag '%s' does not exist", reference))
			return nil, nil, err
		}

		manifestFilter = repositories.NewManifestFilter().ById(tag.GetRepositoryManifestId())
	} else {
		manifestFilter = repositories.NewManifestFilter().ByRepositoryId(repositoryId).ByDigest(reference)
	}

	manifest, err := tx.Manifests().First(ctx, manifestFilter)
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
	repoIdentifier := middlewares.GetRepoIdentifier(ctx)

	err := checkAccess(ctx, repoIdentifier, ociAuthentication.PullAccess)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	reference := vars["reference"]

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, _, repository, err := getRepositoryByIdentifier(ctx, tx, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, blob, err := getManifestByReference(ctx, tx, repository.GetId(), reference)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	blobService := ioc.GetDependency[blobStorage.Service](scope)
	redirectUri, err := blobService.GetBlobDownloadLink(ctx, blob.GetDigest())
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

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, _, repository, err := getRepositoryByIdentifier(ctx, tx, repoIdentifier)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, blob, err := getManifestByReference(ctx, tx, repository.GetId(), reference)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	w.Header().Set("Docker-Content-Digest", blob.GetDigest())
	w.Header().Set("Content-Length", strconv.FormatInt(blob.GetSize(), 10))
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

	dbService := ioc.GetDependency[services.DbService](scope)
	tx, err := dbService.GetTransaction()
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	_, _, repository, err := getRepositoryByIdentifier(ctx, tx, repoIdentifier)
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

	blobs := ioc.GetDependency[blobStorage.Service](scope)
	uploadResponse, err := blobs.UploadCompleteBlob(ctx, digest, bytes.NewReader(bodyBytes), blobStorage.BlobContentTypeManifest)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	blob, err := tx.Blobs().First(ctx, repositories.NewBlobFilter().ByDigest(uploadResponse.Digest))
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}
	if blob == nil {
		blob = repositories.NewBlob(uploadResponse.Digest, int64(len(bodyBytes)))
		if err := tx.Blobs().Insert(ctx, blob); err != nil {
			ociError.HandleHttpError(w, r, err)
			return
		}
	}

	manifest := repositories.NewManifest(repository.GetId(), blob.GetId(), uploadResponse.Digest)
	err = tx.Manifests().Insert(ctx, manifest)
	if err != nil {
		ociError.HandleHttpError(w, r, err)
		return
	}

	vars := mux.Vars(r)
	reference := vars["reference"]

	if strings.HasPrefix(reference, "sha256:") {
		// if it's a reference, check if the digest matches
		if reference != digest {
			ociError.HandleHttpError(w, r, ociError.NewOciError(ociError.DigestInvalid))
		}
	} else {
		// if it's a tag, insert it into the database
		err := tx.Tags().Insert(ctx, repositories.NewTag(repository.GetId(), manifest.GetId(), reference))
		if err != nil {
			ociError.HandleHttpError(w, r, err)
			return
		}
	}

	location := fmt.Sprintf("/v2/%s/%s/manifests/%s", repoIdentifier.ProjectSlug, repoIdentifier.RepositorySlug, uploadResponse.Digest)

	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}
