package decoding

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/the127/dockyard/internal/utils/apiError"
)

func HttpBodyAsJson(w http.ResponseWriter, r *http.Request, v any) error {
	contentTypeHeader := r.Header.Get("Content-Type")
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentTypeHeader, ";")[0]))
	if mediaType != "application/json" {
		return fmt.Errorf("expected application/json, got %s: %w", contentTypeHeader, apiError.ErrApiUnsupportedMediaType)
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(v)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError

		switch {

		case errors.As(err, &syntaxError):
			return fmt.Errorf("invalid JSON syntax at position %d: %w", syntaxError.Offset, apiError.ErrApiBadRequest)

		// https://github.com/golang/go/issues/25956.
		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("invalid JSON syntax: %w", apiError.ErrApiBadRequest)

		case errors.As(err, &unmarshalTypeError):
			return fmt.Errorf("invalid JSON syntax for field %q (at position %d): %w", unmarshalTypeError.Field, unmarshalTypeError.Offset, apiError.ErrApiBadRequest)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			unknownFieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("unknown field %q: %w", unknownFieldName, apiError.ErrApiBadRequest)

		case errors.Is(err, io.EOF):
			return fmt.Errorf("request body is empty: %w", apiError.ErrApiBadRequest)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("request body is too large: %w", apiError.ErrApiBadRequest)

		default:
			return fmt.Errorf("failed to decode request body: %w", err)
		}
	}

	err = ensureNoTrailingData(decoder)
	if err != nil {
		return err
	}

	return nil
}

func ensureNoTrailingData(decoder *json.Decoder) error {
	err := decoder.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return fmt.Errorf("unexpected trailing data: %w", apiError.ErrApiBadRequest)
	}

	return nil
}
