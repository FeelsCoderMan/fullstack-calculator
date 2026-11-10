package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"fullstack-calculator/backend/internal/apperror"
)

// errorResponse is the body of every error response except validation errors:
// a code only, never a message (AGENTS.md Decision 2); the frontend maps codes
// to text.
type errorResponse struct {
	Code apperror.Code `json:"code"`
}

// fieldErrorResponse is one entry of a validation error body, which is a bare
// array of them (api/openapi.yaml ValidationErrorResponse).
type fieldErrorResponse struct {
	Field string        `json:"field"`
	Code  apperror.Code `json:"code"`
}

// failure is what writeError records for RequestLogger: the error as returned
// (its text has the full wrap context), the resolved application error (for
// its stack) and the code the client actually received.
type failure struct {
	err    error
	appErr *apperror.Error
	code   apperror.Code
}

// codeStatus is the single place that maps error codes to HTTP status codes.
// Rule of thumb: 400 = the body can't be read as a JSON object, 422 = it can
// be read, but a field or the math is invalid.
var codeStatus = map[apperror.Code]int{
	apperror.CodeRequestInvalidJSON:          http.StatusBadRequest,
	apperror.CodeValidationFailed:            http.StatusUnprocessableEntity,
	apperror.CodeRequestNotFound:             http.StatusNotFound,
	apperror.CodeRequestMethodNotAllowed:     http.StatusMethodNotAllowed,
	apperror.CodeRequestBodyTooLarge:         http.StatusRequestEntityTooLarge,
	apperror.CodeRequestUnsupportedMediaType: http.StatusUnsupportedMediaType,
	apperror.CodeDivisionByZero:              http.StatusUnprocessableEntity,
	apperror.CodeNegativeSquareRoot:          http.StatusUnprocessableEntity,
	apperror.CodeResultOverflow:              http.StatusUnprocessableEntity,
	apperror.CodeResultUndefined:             http.StatusUnprocessableEntity,
	apperror.CodeInternal:                    http.StatusInternalServerError,
}

// statusFor returns the HTTP status for a code; an unmapped code is a bug and
// is answered as an internal error.
func statusFor(code apperror.Code) int {
	if status, ok := codeStatus[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// writeError ends a failed request: it resolves err to an application error,
// records it for RequestLogger (which logs it, once, in the request record)
// and writes the JSON error response. It does not log anything itself.
func writeError(c *gin.Context, err error) {
	appErr := apperror.From(err)
	code := appErr.Code
	switch {
	case statusFor(code) == http.StatusInternalServerError:
		code = apperror.CodeInternal // never reveal more than "internal" for a 5xx
	case code == apperror.CodeValidationFailed && len(appErr.Fields) == 0:
		code = apperror.CodeInternal // a bug: an empty array would tell the client nothing
	}
	c.Set(contextKeyFailure, failure{err: err, appErr: appErr, code: code})
	c.AbortWithStatusJSON(statusFor(code), errorBody(code, appErr.Fields))
}

// errorBody builds the response body for code: the bare array of field errors
// for a validation error, {"code": ...} for everything else. It returns any
// because the two bodies are different JSON types (array and object).
func errorBody(code apperror.Code, fields []apperror.FieldError) any {
	if code != apperror.CodeValidationFailed {
		return errorResponse{Code: code}
	}
	body := make([]fieldErrorResponse, 0, len(fields))
	for _, f := range fields {
		body = append(body, fieldErrorResponse{Field: f.Field, Code: f.Code})
	}
	return body
}

// recordedFailure returns the failure writeError recorded for this request.
func recordedFailure(c *gin.Context) (failure, bool) {
	v, ok := c.Get(contextKeyFailure)
	if !ok {
		return failure{}, false
	}
	f, ok := v.(failure)
	return f, ok
}

// notFound answers unknown routes (gin NoRoute).
func notFound(c *gin.Context) { writeError(c, apperror.ErrNotFound) }

// methodNotAllowed answers a known path with the wrong method (gin NoMethod).
func methodNotAllowed(c *gin.Context) { writeError(c, apperror.ErrMethodNotAllowed) }
