package apperror

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// Error is an application error: a client-facing Code plus, for the logs, an
// optional wrapped cause and (for unexpected failures) the stack where it was
// created. Error text is developer context and is never sent to clients.
type Error struct {
	Code Code
	// Fields lists per-field problems; set only for CodeValidationFailed,
	// whose response body is this list.
	Fields []FieldError

	cause error
	stack []uintptr
}

// FieldError is one problem with one request field.
type FieldError struct {
	// Field is the JSON property name as sent by the client.
	Field string
	Code  Code
}

// Sentinels for expected failures. They carry no stack: the code and the
// caller's wrap context (fmt.Errorf("...: %w", ErrX)) explain them fully.
// Compare with errors.Is, which matches by code.
var (
	ErrInvalidJSON          = &Error{Code: CodeRequestInvalidJSON}
	ErrValidationFailed     = &Error{Code: CodeValidationFailed}
	ErrNotFound             = &Error{Code: CodeRequestNotFound}
	ErrMethodNotAllowed     = &Error{Code: CodeRequestMethodNotAllowed}
	ErrBodyTooLarge         = &Error{Code: CodeRequestBodyTooLarge}
	ErrUnsupportedMediaType = &Error{Code: CodeRequestUnsupportedMediaType}
	ErrDivisionByZero       = &Error{Code: CodeDivisionByZero}
	ErrNegativeSquareRoot   = &Error{Code: CodeNegativeSquareRoot}
	ErrResultOverflow       = &Error{Code: CodeResultOverflow}
	ErrResultUndefined      = &Error{Code: CodeResultUndefined}
	ErrInternal             = &Error{Code: CodeInternal}
)

// ErrPanic is the cause of an error built from a recovered panic.
var ErrPanic = errors.New("panic")

// Error returns the code, followed by the cause when there is one.
func (e *Error) Error() string {
	if e.cause == nil {
		return string(e.Code)
	}
	return fmt.Sprintf(errorWithCauseFormat, e.Code, e.cause)
}

// Unwrap returns the cause, so errors.Is and errors.As see through an Error.
func (e *Error) Unwrap() error { return e.cause }

// Is reports whether target is an *Error with the same code, so
// errors.Is(err, ErrDivisionByZero) holds however err was built or wrapped.
func (e *Error) Is(target error) bool {
	var t *Error
	return errors.As(target, &t) && t.Code == e.Code
}

// Stack returns the captured call stack as "<pkg>.<func> <dir>/<file>:<line>"
// frames, outermost call last. It is empty for expected failures.
func (e *Error) Stack() []string {
	if len(e.stack) == 0 {
		return nil
	}
	frames := runtime.CallersFrames(e.stack)
	out := make([]string, 0, len(e.stack))
	for {
		f, more := frames.Next()
		out = append(out, fmt.Sprintf(stackFrameFormat, shortFunc(f.Function), shortPath(f.File), f.Line))
		if !more {
			return out
		}
	}
}

// Internal wraps an unexpected error as CodeInternal and captures the stack at
// the call site. Call it where the unexpected error is first seen, so the
// stack points at its origin.
func Internal(err error) *Error {
	return &Error{Code: CodeInternal, cause: err, stack: capture()}
}

// Panic builds a CodeInternal error from a recovered panic value. Call it in
// the deferred function that recovered, so the stack includes the panic site.
func Panic(v any) *Error {
	return &Error{Code: CodeInternal, cause: fmt.Errorf(panicFormat, ErrPanic, v), stack: capture()}
}

// Validation builds a CodeValidationFailed error listing every field problem.
func Validation(fields ...FieldError) *Error {
	return &Error{Code: CodeValidationFailed, Fields: fields}
}

// From returns the *Error in err's chain. An error without one is unexpected
// and becomes Internal(err), with the stack captured here. From(nil) is nil.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	if appErr, ok := errors.AsType[*Error](err); ok {
		return appErr
	}
	return &Error{Code: CodeInternal, cause: err, stack: capture()}
}

func capture() []uintptr {
	pcs := make([]uintptr, maxStackFrames)
	return pcs[:runtime.Callers(skipToCaller, pcs)]
}

// shortFunc drops the import path from a function name:
// "fullstack-calculator/backend/internal/service.(*Calc).Divide" becomes
// "service.(*Calc).Divide".
func shortFunc(name string) string {
	return name[strings.LastIndex(name, importPathSeparator)+1:]
}

// shortPath keeps only the last directory and the file name, so build paths
// (a developer's home directory, CI workspaces) never reach the logs.
func shortPath(file string) string {
	return filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file))
}
