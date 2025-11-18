package ociError

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/the127/dockyard/internal/args"
	"github.com/the127/dockyard/internal/logging"
)

type OciErrorCode string

const (
	// BlobUnknown code-1: blob unknown to registry
	BlobUnknown OciErrorCode = "BLOB_UNKNOWN"

	// BlobUploadInvalid code-2: blob upload invalid
	BlobUploadInvalid OciErrorCode = "BLOB_UPLOAD_INVALID"

	// BlobUploadUnknown code-3: blob upload unknown to registry
	BlobUploadUnknown OciErrorCode = "BLOB_UPLOAD_UNKNOWN"

	// DigestInvalid code-4: provided digest did not match uploaded content
	DigestInvalid OciErrorCode = "DIGEST_INVALID"

	// ManifestBlobUnknown code-5: manifest references a manifest or blob unknown to registry
	ManifestBlobUnknown OciErrorCode = "MANIFEST_BLOB_UNKNOWN"

	// ManifestInvalid code-6: manifest invalid
	ManifestInvalid OciErrorCode = "MANIFEST_INVALID"

	// ManifestUnknown code-7: manifest unknown to registry
	ManifestUnknown OciErrorCode = "MANIFEST_UNKNOWN"

	// NameInvalid code-8: invalid repository name
	NameInvalid OciErrorCode = "NAME_INVALID"

	// NameUnknown code-9: repository name not known to registry
	NameUnknown OciErrorCode = "NAME_UNKNOWN"

	// SizeInvalid code-10: provided length did not match content length
	SizeInvalid OciErrorCode = "SIZE_INVALID"

	// Unauthorized code-11: authentication required
	Unauthorized OciErrorCode = "UNAUTHORIZED"

	// Denied code-12: requested access to the resource is denied
	Denied OciErrorCode = "DENIED"

	// Unsupported code-13: the operation is unsupported
	Unsupported OciErrorCode = "UNSUPPORTED"

	// TooManyRequests code-14: too many requests
	TooManyRequests OciErrorCode = "TOO_MANY_REQUESTS"
)

type OciError struct {
	HttpCode int          `json:"-"`
	Code     OciErrorCode `json:"code"`
	Message  string       `json:"message,omitempty"`
	Headers  map[string]string
}

func NewOciError(code OciErrorCode) *OciError {
	return &OciError{
		HttpCode: http.StatusBadRequest,
		Code:     code,
		Headers:  make(map[string]string),
	}
}

func (e *OciError) WithMessage(message string) *OciError {
	e.Message = message
	return e
}

func (e *OciError) WithHttpCode(httpCode int) *OciError {
	e.HttpCode = httpCode
	return e
}

func (e *OciError) WithHeader(key, value string) *OciError {
	e.Headers[key] = value
	return e
}

func (e *OciError) Error() string {
	if e.Message == "" {
		return string(e.Code)
	}

	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type Wrapper struct {
	Errors []*OciError `json:"errors"`
}

func HandleHttpError(w http.ResponseWriter, r *http.Request, err error) {
	var message string
	var ociError *OciError

	if errors.As(err, &ociError) {
		wrapper := Wrapper{
			Errors: []*OciError{ociError},
		}

		for k, v := range ociError.Headers {
			w.Header().Set(k, v)
		}

		logging.Logger.Errorf("HTTP Error: %d %s", ociError.HttpCode, ociError.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(ociError.HttpCode)

		if r.Method != http.MethodHead {
			err = json.NewEncoder(w).Encode(wrapper)
			if err == nil {
				return
			}
		} else {
			return
		}
	}

	if err != nil {
		if args.IsProduction() {
			message = "Internal Server Error"
		} else {
			message = err.Error()
		}

		logging.Logger.Errorf("HTTP Error: %d %s", http.StatusInternalServerError, message)
		w.WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(w).Encode(message)
		if err == nil {
			return
		}
	}

	if err != nil {
		if args.IsProduction() {
			message = "Internal Server Error"
		} else {
			message = err.Error()
		}

		logging.Logger.Errorf("HTTP Error: %d %s", http.StatusInternalServerError, message)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}
}
