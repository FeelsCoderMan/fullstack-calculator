package apperror

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var errDB = errors.New("connection refused")

func TestFrom(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCode  Code
		wantText  string
		wantStack bool
	}{
		{
			name:     "sentinel as is",
			err:      ErrDivisionByZero,
			wantCode: CodeDivisionByZero,
			wantText: "error.calculation.division.by.zero",
		},
		{
			name:     "sentinel wrapped with context keeps its code",
			err:      fmt.Errorf("divide 1 by 0: %w", ErrDivisionByZero),
			wantCode: CodeDivisionByZero,
			wantText: "error.calculation.division.by.zero",
		},
		{
			name:      "explicit internal error keeps cause and stack",
			err:       fmt.Errorf("load: %w", Internal(errDB)),
			wantCode:  CodeInternal,
			wantText:  "error.internal: connection refused",
			wantStack: true,
		},
		{
			name:      "unknown error becomes internal with a stack",
			err:       errDB,
			wantCode:  CodeInternal,
			wantText:  "error.internal: connection refused",
			wantStack: true,
		},
		{
			name:     "validation error keeps its fields",
			err:      Validation(FieldError{Field: "a", Code: CodeFieldRequired}),
			wantCode: CodeValidationFailed,
			wantText: "error.validation.failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := From(tt.err)
			if got.Code != tt.wantCode {
				t.Errorf("From(%v).Code = %q, want %q", tt.err, got.Code, tt.wantCode)
			}
			if got.Error() != tt.wantText {
				t.Errorf("From(%v).Error() = %q, want %q", tt.err, got.Error(), tt.wantText)
			}
			if hasStack := len(got.Stack()) > 0; hasStack != tt.wantStack {
				t.Errorf("From(%v) has stack = %v, want %v", tt.err, hasStack, tt.wantStack)
			}
		})
	}
}

func TestFromNil(t *testing.T) {
	if got := From(nil); got != nil {
		t.Errorf("From(nil) = %v, want nil", got)
	}
}

func TestIsMatchesByCode(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "same sentinel",
			err:    ErrNotFound,
			target: ErrNotFound,
			want:   true,
		},
		{
			name:   "wrapped sentinel",
			err:    fmt.Errorf("route: %w", ErrNotFound),
			target: ErrNotFound,
			want:   true,
		},
		{
			name:   "fresh validation error matches the sentinel",
			err:    Validation(FieldError{Field: "b", Code: CodeFieldInvalidType}),
			target: ErrValidationFailed,
			want:   true,
		},
		{
			name:   "internal error matches ErrInternal",
			err:    Internal(errDB),
			target: ErrInternal,
			want:   true,
		},
		{
			name:   "internal error still exposes its cause",
			err:    Internal(errDB),
			target: errDB,
			want:   true,
		},
		{
			name:   "different code",
			err:    ErrDivisionByZero,
			target: ErrNegativeSquareRoot,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.err, tt.target, got, tt.want)
			}
		})
	}
}

func TestStackPointsAtCallSite(t *testing.T) {
	tests := []struct {
		name      string
		build     func() *Error
		wantFirst string
	}{
		{
			name:      "Internal",
			build:     func() *Error { return Internal(errDB) },
			wantFirst: "apperror.TestStackPointsAtCallSite.func1 apperror/errors_test.go:",
		},
		{
			name: "Panic in a deferred recover",
			build: func() (appErr *Error) {
				defer func() { appErr = Panic(recover()) }()
				panic("boom")
			},
			wantFirst: "apperror.TestStackPointsAtCallSite.func2.1 apperror/errors_test.go:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := tt.build().Stack()
			if len(stack) == 0 || !strings.HasPrefix(stack[0], tt.wantFirst) {
				t.Fatalf("%s stack[0] = %q, want prefix %q (stack: %q)", tt.name, first(stack), tt.wantFirst, stack)
			}
			for _, frame := range stack {
				if strings.Count(frame, "/") > 2 || strings.Contains(frame, "home") {
					t.Errorf("frame %q leaks a long or local path", frame)
				}
			}
		})
	}
}

func TestPanicCause(t *testing.T) {
	got := func() (appErr *Error) {
		defer func() { appErr = Panic(recover()) }()
		panic("boom")
	}()

	if got.Code != CodeInternal {
		t.Errorf("Panic(\"boom\").Code = %q, want %q", got.Code, CodeInternal)
	}
	if !errors.Is(got, ErrPanic) {
		t.Errorf("errors.Is(Panic(\"boom\"), ErrPanic) = false, want true")
	}
	if want := "error.internal: panic: boom"; got.Error() != want {
		t.Errorf("Panic(\"boom\").Error() = %q, want %q", got.Error(), want)
	}
}

func TestValidationFields(t *testing.T) {
	fields := []FieldError{
		{Field: "a", Code: CodeFieldRequired},
		{Field: "b", Code: CodeFieldInvalidType},
	}
	got := Validation(fields...)
	if diff := cmp.Diff(fields, got.Fields); diff != "" {
		t.Errorf("Validation(...).Fields mismatch (-want +got):\n%s", diff)
	}
}

func TestCodesAreDisjointAndUnique(t *testing.T) {
	seen := map[Code]bool{}
	for _, c := range append(Codes(), FieldCodes()...) {
		if seen[c] {
			t.Errorf("code %q is listed twice", c)
		}
		seen[c] = true
		if !strings.HasPrefix(string(c), "error.") || strings.ToLower(string(c)) != string(c) {
			t.Errorf("code %q must be lowercase and start with \"error.\"", c)
		}
	}
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
