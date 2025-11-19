package apiError

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/the127/dockyard/internal/args"
	"github.com/the127/dockyard/internal/logging"
)

var ErrApiBadRequest = errors.New("bad Request")
var ErrApiUnsupportedMediaType = errors.New("unsupported media type")

var ErrApiNotFound = errors.New("not found")
var ErrApiTenantNotFound = fmt.Errorf("tenant not found: %w", ErrApiNotFound)
var ErrApiProjectNotFound = fmt.Errorf("project not found: %w", ErrApiNotFound)
var ErrApiUserNotFound = fmt.Errorf("user not found: %w", ErrApiNotFound)
var ErrApiRepositoryNotFound = fmt.Errorf("repository not found: %w", ErrApiNotFound)
var ErrApiManifestNotFound = fmt.Errorf("manifest not found: %w", ErrApiNotFound)
var ErrApiTagNotFound = fmt.Errorf("tag not found: %w", ErrApiNotFound)
var ErrApiBlobNotFound = fmt.Errorf("blob not found: %w", ErrApiNotFound)
var ErrApiRepositoryBlobNotFound = fmt.Errorf("repository blob not found: %w", ErrApiNotFound)
var ErrApiFileNotFound = fmt.Errorf("file not found: %w", ErrApiNotFound)
var ErrApiPatNotFound = fmt.Errorf("pat not found: %w", ErrApiNotFound)

var ErrApiUnauthorized = errors.New("unauthorized")

func HandleHttpError(w http.ResponseWriter, err error) {
	var code int
	var message string

	switch {
	case errors.Is(err, ErrApiBadRequest):
		code = http.StatusBadRequest
		message = err.Error()

	case errors.Is(err, ErrApiNotFound):
		code = http.StatusNotFound
		message = err.Error()

	case errors.Is(err, ErrApiUnsupportedMediaType):
		code = http.StatusUnsupportedMediaType
		message = err.Error()

	case errors.Is(err, ErrApiUnauthorized):
		code = http.StatusUnauthorized
		message = err.Error()

	default:
		code = http.StatusInternalServerError
		if args.IsProduction() {
			message = "Internal Server Error"
		} else {
			message = err.Error()
		}
	}

	logging.Logger.Errorf("HTTP Error: %d %s", code, message)
	http.Error(w, message, code)
}
