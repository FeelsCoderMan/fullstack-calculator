package handler

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"fullstack-calculator/backend/internal/apperror"
)

func TestStatusFor(t *testing.T) {
	tests := []struct {
		name string
		code apperror.Code
		want int
	}{
		{name: "invalid json", code: apperror.CodeRequestInvalidJSON, want: http.StatusBadRequest},
		{name: "validation failed", code: apperror.CodeValidationFailed, want: http.StatusUnprocessableEntity},
		{name: "not found", code: apperror.CodeRequestNotFound, want: http.StatusNotFound},
		{name: "method not allowed", code: apperror.CodeRequestMethodNotAllowed, want: http.StatusMethodNotAllowed},
		{name: "body too large", code: apperror.CodeRequestBodyTooLarge, want: http.StatusRequestEntityTooLarge},
		{name: "unsupported media type", code: apperror.CodeRequestUnsupportedMediaType, want: http.StatusUnsupportedMediaType},
		{name: "division by zero", code: apperror.CodeDivisionByZero, want: http.StatusUnprocessableEntity},
		{name: "negative square root", code: apperror.CodeNegativeSquareRoot, want: http.StatusUnprocessableEntity},
		{name: "result overflow", code: apperror.CodeResultOverflow, want: http.StatusUnprocessableEntity},
		{name: "result undefined", code: apperror.CodeResultUndefined, want: http.StatusUnprocessableEntity},
		{name: "internal", code: apperror.CodeInternal, want: http.StatusInternalServerError},
		{name: "unmapped code is internal", code: apperror.CodeFieldRequired, want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusFor(tt.code); got != tt.want {
				t.Errorf("statusFor(%q) = %d, want %d", tt.code, got, tt.want)
			}
		})
	}
}

// TestEveryCodeHasAStatus fails when a top-level code is added to apperror
// without a status mapping (it would silently become a 500).
func TestEveryCodeHasAStatus(t *testing.T) {
	for _, code := range apperror.Codes() {
		if _, ok := codeStatus[code]; !ok {
			t.Errorf("apperror code %q has no entry in codeStatus", code)
		}
	}
}

// TestCodesMatchSpec keeps the error codes in apperror and api/openapi.yaml
// identical, so the frontend's generated types know every code we send.
func TestCodesMatchSpec(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		codes  []apperror.Code
	}{
		{
			name:   "top-level codes",
			schema: "ErrorCode",
			codes:  apperror.Codes(),
		},
		{
			name:   "field codes",
			schema: "FieldErrorCode",
			codes:  apperror.FieldCodes(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, ok := loadSpec(t).Components.Schemas[tt.schema]
			if !ok {
				t.Fatalf("api/openapi.yaml has no schema %q", tt.schema)
			}
			var documented []string
			for _, v := range ref.Value.Enum {
				s, _ := v.(string)
				documented = append(documented, s)
			}
			var defined []string
			for _, c := range tt.codes {
				defined = append(defined, string(c))
			}
			if diff := cmp.Diff(documented, defined); diff != "" {
				t.Errorf("%s enum and apperror disagree (-spec +apperror):\n%s", tt.schema, diff)
			}
		})
	}
}
