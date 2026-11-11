package handler

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"fullstack-calculator/backend/internal/apperror"
	"fullstack-calculator/backend/internal/service"
)

// requestTypes are every request body type bound with bindJSON.
var requestTypes = []reflect.Type{
	reflect.TypeFor[binaryRequest](),
	reflect.TypeFor[sqrtRequest](),
}

// bindingRules returns the rule names and params of a field's binding tag,
// e.g. "min=0" → ("min", "0").
func bindingRules(t *testing.T, f reflect.StructField) map[string]string {
	t.Helper()
	rules := map[string]string{}
	for rule := range strings.SplitSeq(f.Tag.Get(tagKeyBinding), jsonTagSeparator) {
		name, param, _ := strings.Cut(rule, ruleParamSep)
		rules[name] = param
	}
	return rules
}

// TestValidationTagsHaveCodes catches a binding rule that would fail without
// a field code to report (fieldErrors answers that as a 500).
func TestValidationTagsHaveCodes(t *testing.T) {
	for _, typ := range requestTypes {
		for i := range typ.NumField() {
			f := typ.Field(i)
			for rule := range bindingRules(t, f) {
				if rule == tagOmitEmpty {
					continue
				}
				if _, ok := validationTagCodes[rule]; !ok {
					t.Errorf("%s.%s: rule %q has no entry in validationTagCodes", typ.Name(), f.Name, rule)
				}
			}
		}
	}
}

// TestRequestFieldsAreNumbers guards bindJSON's assumption: a field of
// another type would fail json.Unmarshal on a wrong-typed value (a 500 instead
// of a 422 invalid.type), and a field without a json name would never be bound.
func TestRequestFieldsAreNumbers(t *testing.T) {
	for _, typ := range requestTypes {
		for i := range typ.NumField() {
			f := typ.Field(i)
			if f.Type != reflect.TypeFor[number]() || jsonName(f) == "" {
				t.Errorf("%s.%s: type %v, json name %q; want number with a json name", typ.Name(), f.Name, f.Type, jsonName(f))
			}
		}
	}
}

// TestPrecisionBoundsMatchService keeps the min/max in the precision tags
// equal to the service limits.
func TestPrecisionBoundsMatchService(t *testing.T) {
	tests := []struct {
		name string
		rule string
		want int
	}{
		{name: "lower bound", rule: tagMin, want: service.MinPrecision},
		{name: "upper bound", rule: tagMax, want: service.MaxPrecision},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, typ := range requestTypes {
				f, ok := typ.FieldByName("Precision")
				if !ok {
					t.Fatalf("%s has no Precision field", typ.Name())
				}
				got := bindingRules(t, f)[tt.rule]
				if got != strconv.Itoa(tt.want) {
					t.Errorf("%s.Precision %s = %q, want %d", typ.Name(), tt.rule, got, tt.want)
				}
			}
		})
	}
}

// TestBindJSONMatchesNamesExactly pins down that property names are
// case-sensitive, although encoding/json alone matches them case-insensitively.
func TestBindJSONMatchesNamesExactly(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "upper-case operand name is unknown, not the operand",
			path:       pathAdd,
			body:       `{"LEFTOPERAND":1,"rightOperand":2}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   `[{"field":"leftOperand","code":"error.field.required"},{"field":"LEFTOPERAND","code":"error.field.unknown"}]`,
		},
		{
			name:       "differently cased precision is unknown and does not round",
			path:       pathSqrt,
			body:       `{"operand":2,"Precision":2}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   `[{"field":"Precision","code":"error.field.unknown"}]`,
		},
		{
			name:       "exact names bind",
			path:       pathSqrt,
			body:       `{"operand":2,"precision":2}`,
			wantStatus: http.StatusOK,
			wantBody:   `{"result":1.41}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rec := serveJSON(t, newTestRouter(t), tt.path, binding.MIMEJSON, tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("POST %s %s: status = %d, want %d", tt.path, tt.body, rec.Code, tt.wantStatus)
			}
			assertJSONEqual(t, tt.wantBody, rec.Body.Bytes())
			assertMatchesSpec(t, req, rec)
		})
	}
}

// uncodedRequest uses a binding rule, nocode, that has no field code.
type uncodedRequest struct {
	Field number `json:"field" binding:"nocode"`
}

// TestFieldErrorsFailsAsInternal covers the bug paths of fieldErrors: they
// must end as a 500, never as a field error the client can't interpret.
func TestFieldErrorsFailsAsInternal(t *testing.T) {
	validate, err := newValidator()
	if err != nil {
		t.Fatalf("newValidator() error = %v", err)
	}
	ruleWithoutCode := "nocode" // matches the tag of uncodedRequest
	if err := validate.RegisterValidation(ruleWithoutCode, func(validator.FieldLevel) bool { return false }); err != nil {
		t.Fatalf("RegisterValidation(%q) error = %v", ruleWithoutCode, err)
	}

	tests := []struct {
		name      string
		in        error
		wantCause error
	}{
		{
			name:      "rule without a field code",
			in:        validate.Struct(uncodedRequest{}),
			wantCause: errNoFieldCode,
		},
		{
			name: "not a validation error",
			in:   validate.Struct(42),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fieldErrors(tt.in)

			if got != nil {
				t.Errorf("fieldErrors(%v) = %v, want nil", tt.in, got)
			}
			if !errors.Is(err, apperror.ErrInternal) {
				t.Errorf("fieldErrors(%v) error = %v, want %v", tt.in, err, apperror.ErrInternal)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("fieldErrors(%v) error = %v, want %v", tt.in, err, tt.wantCause)
			}
		})
	}
}
