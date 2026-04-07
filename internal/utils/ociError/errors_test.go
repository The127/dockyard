package ociError

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/the127/dockyard/internal/args"
	"github.com/the127/dockyard/internal/logging"
)

type OciErrorTestSuite struct {
	suite.Suite
}

func TestOciErrorTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(OciErrorTestSuite))
}

func (s *OciErrorTestSuite) SetupSuite() {
	logging.Init()
}

// NewOciError

func (s *OciErrorTestSuite) TestNewOciError_DefaultHttpCode() {
	// act
	err := NewOciError(BlobUnknown)

	// assert
	s.Equal(http.StatusBadRequest, err.HttpCode)
}

func (s *OciErrorTestSuite) TestNewOciError_CodeIsSet() {
	// act
	err := NewOciError(ManifestUnknown)

	// assert
	s.Equal(ManifestUnknown, err.Code)
}

func (s *OciErrorTestSuite) TestNewOciError_HeadersMapInitialized() {
	// act
	err := NewOciError(BlobUnknown)

	// assert
	s.NotNil(err.Headers)
}

// Builder methods

func (s *OciErrorTestSuite) TestWithMessage_ReturnsSelf() {
	// arrange
	err := NewOciError(BlobUnknown)

	// act
	result := err.WithMessage("some message")

	// assert
	s.Same(err, result)
}

func (s *OciErrorTestSuite) TestWithMessage_SetsMessage() {
	// act
	err := NewOciError(BlobUnknown).WithMessage("test message")

	// assert
	s.Equal("test message", err.Message)
}

func (s *OciErrorTestSuite) TestWithHttpCode_ReturnsSelf() {
	// arrange
	err := NewOciError(BlobUnknown)

	// act
	result := err.WithHttpCode(http.StatusNotFound)

	// assert
	s.Same(err, result)
}

func (s *OciErrorTestSuite) TestWithHttpCode_SetsHttpCode() {
	// act
	err := NewOciError(BlobUnknown).WithHttpCode(http.StatusNotFound)

	// assert
	s.Equal(http.StatusNotFound, err.HttpCode)
}

func (s *OciErrorTestSuite) TestWithHeader_ReturnsSelf() {
	// arrange
	err := NewOciError(BlobUnknown)

	// act
	result := err.WithHeader("X-Foo", "bar")

	// assert
	s.Same(err, result)
}

func (s *OciErrorTestSuite) TestWithHeader_SetsHeader() {
	// act
	err := NewOciError(BlobUnknown).WithHeader("X-Foo", "bar")

	// assert
	s.Equal("bar", err.Headers["X-Foo"])
}

// Error()

func (s *OciErrorTestSuite) TestError_WithoutMessage() {
	// act
	err := NewOciError(NameUnknown)

	// assert
	s.Equal("NAME_UNKNOWN", err.Error())
}

func (s *OciErrorTestSuite) TestError_WithMessage() {
	// act
	err := NewOciError(NameUnknown).WithMessage("repo not found")

	// assert
	s.Equal("NAME_UNKNOWN: repo not found", err.Error())
}

// HandleHttpError — *OciError

func (s *OciErrorTestSuite) TestHandleHttpError_OciError_StatusCode() {
	// arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ociErr := NewOciError(ManifestUnknown).WithHttpCode(http.StatusNotFound)

	// act
	HandleHttpError(w, r, ociErr)

	// assert
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *OciErrorTestSuite) TestHandleHttpError_OciError_ContentTypeHeader() {
	// arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ociErr := NewOciError(ManifestUnknown)

	// act
	HandleHttpError(w, r, ociErr)

	// assert
	s.Equal("application/json", w.Header().Get("Content-Type"))
}

func (s *OciErrorTestSuite) TestHandleHttpError_OciError_JsonBodyShape() {
	// arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ociErr := NewOciError(DigestInvalid).WithMessage("bad digest")

	// act
	HandleHttpError(w, r, ociErr)

	// assert
	var wrapper Wrapper
	decodeErr := json.NewDecoder(w.Body).Decode(&wrapper)
	s.NoError(decodeErr)
	s.Len(wrapper.Errors, 1)
	s.Equal(DigestInvalid, wrapper.Errors[0].Code)
	s.Equal("bad digest", wrapper.Errors[0].Message)
}

func (s *OciErrorTestSuite) TestHandleHttpError_OciError_CustomHeadersForwarded() {
	// arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ociErr := NewOciError(Unauthorized).
		WithHttpCode(http.StatusUnauthorized).
		WithHeader("WWW-Authenticate", "Bearer realm=\"test\"")

	// act
	HandleHttpError(w, r, ociErr)

	// assert
	s.Equal("Bearer realm=\"test\"", w.Header().Get("WWW-Authenticate"))
}

// HandleHttpError — HEAD request with *OciError

func (s *OciErrorTestSuite) TestHandleHttpError_OciError_HeadRequest_StatusCode() {
	// arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodHead, "/", nil)
	ociErr := NewOciError(ManifestUnknown).WithHttpCode(http.StatusNotFound)

	// act
	HandleHttpError(w, r, ociErr)

	// assert
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *OciErrorTestSuite) TestHandleHttpError_OciError_HeadRequest_NoBody() {
	// arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodHead, "/", nil)
	ociErr := NewOciError(ManifestUnknown).WithHttpCode(http.StatusNotFound)

	// act
	HandleHttpError(w, r, ociErr)

	// assert
	s.Equal(0, w.Body.Len())
}

// HandleHttpError — plain error (non-production)
// args.IsProduction() returns false when the environment var has not been set
// via args.Init(), which is the case in unit tests.

func (s *OciErrorTestSuite) TestHandleHttpError_PlainError_NonProduction_StatusCode() {
	// arrange — IsProduction() is false because args.Init() is never called in tests
	s.False(args.IsProduction())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// act
	HandleHttpError(w, r, errors.New("something broke"))

	// assert
	s.Equal(http.StatusInternalServerError, w.Code)
}

func (s *OciErrorTestSuite) TestHandleHttpError_PlainError_NonProduction_BodyContainsMessage() {
	// arrange — IsProduction() is false because args.Init() is never called in tests
	s.False(args.IsProduction())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// act
	HandleHttpError(w, r, errors.New("something broke"))

	// assert
	var body string
	decodeErr := json.NewDecoder(w.Body).Decode(&body)
	s.NoError(decodeErr)
	s.Equal("something broke", body)
}

// HandleHttpError — nil error is a no-op

func (s *OciErrorTestSuite) TestHandleHttpError_NilError_NoWrite() {
	// arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// act
	HandleHttpError(w, r, nil)

	// assert
	s.Equal(http.StatusOK, w.Code) // httptest default, no WriteHeader called
	s.Equal(0, w.Body.Len())
}
