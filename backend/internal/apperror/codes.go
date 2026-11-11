// Package apperror is the application's error catalog. It gives every expected
// failure a stable, client-facing Code and carries the developer-facing cause
// (and, for unexpected failures, a stack trace) for the logs.
//
// It knows nothing about HTTP: the handler maps codes to status codes in one
// place. See docs/logging.md for how errors flow into the single request log
// record.
package apperror

// Code is a stable, machine-readable error identifier sent to clients. The
// frontend maps it to a localized message. Codes are API contract: they are
// mirrored in api/openapi.yaml and must never be renamed.
type Code string

// Top-level codes: the "code" of an error response.
const (
	CodeRequestInvalidJSON          Code = "error.request.invalid.json"
	CodeRequestNotFound             Code = "error.request.not.found"
	CodeRequestMethodNotAllowed     Code = "error.request.method.not.allowed"
	CodeRequestBodyTooLarge         Code = "error.request.body.too.large"
	CodeRequestUnsupportedMediaType Code = "error.request.unsupported.media.type"
	CodeDivisionByZero              Code = "error.calculation.division.by.zero"
	CodeZeroNegativePower           Code = "error.calculation.zero.negative.power"
	CodeNegativeSquareRoot          Code = "error.calculation.negative.square.root"
	CodeResultOverflow              Code = "error.calculation.result.overflow"
	CodeResultUndefined             Code = "error.calculation.result.undefined"
	CodeInternal                    Code = "error.internal"
)

// CodeValidationFailed marks an Error that carries field errors. It is
// internal: it selects the 422 status and appears as the "code" of the request
// log record, but it is never sent. The response body is the bare array of
// field errors (api/openapi.yaml ValidationErrorResponse).
const CodeValidationFailed Code = "error.validation.failed"

// Field-level codes: the "code" of an entry in a validation error response,
// which is a bare array of {"field", "code"} (api/openapi.yaml
// ValidationErrorResponse).
const (
	CodeFieldRequired    Code = "error.field.required"
	CodeFieldInvalidType Code = "error.field.invalid.type"
	CodeFieldOutOfRange  Code = "error.field.out.of.range"
	CodeFieldUnknown     Code = "error.field.unknown"
)

// Codes returns every top-level code sent to clients, in catalog order. It
// excludes the internal CodeValidationFailed.
func Codes() []Code {
	return []Code{
		CodeRequestInvalidJSON,
		CodeRequestNotFound,
		CodeRequestMethodNotAllowed,
		CodeRequestBodyTooLarge,
		CodeRequestUnsupportedMediaType,
		CodeDivisionByZero,
		CodeZeroNegativePower,
		CodeNegativeSquareRoot,
		CodeResultOverflow,
		CodeResultUndefined,
		CodeInternal,
	}
}

// FieldCodes returns every field-level code, in catalog order.
func FieldCodes() []Code {
	return []Code{
		CodeFieldRequired,
		CodeFieldInvalidType,
		CodeFieldOutOfRange,
		CodeFieldUnknown,
	}
}
