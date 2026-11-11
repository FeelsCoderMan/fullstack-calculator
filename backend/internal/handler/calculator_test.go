package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin/binding"

	"fullstack-calculator/backend/internal/apperror"
	"fullstack-calculator/backend/internal/service"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantBody    string
	}{
		// Success.
		{
			name:        "adds integers",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":3}`,
		},
		{
			name:        "adds negative decimals",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-1.5,"rightOperand":-2.25}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":-3.75}`,
		},
		{
			name:        "keeps float noise without precision",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0.1,"rightOperand":0.2}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.30000000000000004}`,
		},
		{
			name:        "rounds to precision",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0.1,"rightOperand":0.2,"precision":2}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.3}`,
		},
		{
			name:        "precision zero rounds half away from zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":2,"rightOperand":0.5,"precision":0}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":3}`,
		},
		{
			name:        "precision 2.0 counts as an integer",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1.005,"rightOperand":0,"precision":2.0}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.01}`,
		},
		{
			name:        "null precision means no rounding",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0.1,"rightOperand":0.2,"precision":null}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.30000000000000004}`,
		},
		{
			name:        "exponent notation operands",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e308,"rightOperand":-1E308}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0}`,
		},
		{
			name:        "operand below float64 range underflows to zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e-400,"rightOperand":0}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0}`,
		},
		{
			name:        "integer above 2^53 loses precision without error",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":9007199254740993,"rightOperand":0}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":9007199254740992}`,
		},
		{
			name:        "precision at the upper bound",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0.1,"rightOperand":0.2,"precision":15}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.3}`,
		},
		{
			name:        "precision negative zero counts as zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1.25,"rightOperand":1.25,"precision":-0}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":3}`,
		},
		{
			name:        "content type in upper case",
			contentType: "Application/JSON",
			body:        `{"leftOperand":1,"rightOperand":2}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":3}`,
		},
		{
			name:        "content type with charset",
			contentType: "application/json; charset=utf-8",
			body:        `{"leftOperand":1,"rightOperand":2}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":3}`,
		},
		{
			name:        "whitespace after the object",
			contentType: binding.MIMEJSON,
			body:        "{\"leftOperand\":1,\"rightOperand\":2}\n\t ",
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":3}`,
		},

		// Math errors.
		{
			name:        "positive overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1.7e308,"rightOperand":1.7e308}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "negative overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-1.7e308,"rightOperand":-1.7e308}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},

		// Field errors.
		{
			name:        "both operands missing",
			contentType: binding.MIMEJSON,
			body:        `{}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.required"},{"field":"rightOperand","code":"error.field.required"}]`,
		},
		{
			name:        "null operand is missing, zero is not",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0,"rightOperand":null}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"rightOperand","code":"error.field.required"}]`,
		},
		{
			name:        "numeric string and boolean are not numbers",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":"5","rightOperand":true}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.invalid.type"},{"field":"rightOperand","code":"error.field.invalid.type"}]`,
		},
		{
			name:        "object and array are not numbers",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":{},"rightOperand":[1]}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.invalid.type"},{"field":"rightOperand","code":"error.field.invalid.type"}]`,
		},
		{
			name:        "operands beyond float64 range in both directions",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e309,"rightOperand":-1e309}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.out.of.range"},{"field":"rightOperand","code":"error.field.out.of.range"}]`,
		},
		{
			name:        "precision is a string",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2,"precision":"2"}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"precision","code":"error.field.invalid.type"}]`,
		},
		{
			name:        "precision is not an integer",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2,"precision":2.5}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"precision","code":"error.field.invalid.type"}]`,
		},
		{
			name:        "precision below zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2,"precision":-1}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"precision","code":"error.field.out.of.range"}]`,
		},
		{
			name:        "precision above fifteen",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2,"precision":16}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"precision","code":"error.field.out.of.range"}]`,
		},
		{
			name:        "precision beyond float64 range",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2,"precision":1e400}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"precision","code":"error.field.out.of.range"}]`,
		},
		{
			name:        "unknown properties are reported alphabetically",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2,"zeta":1,"a":2}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"a","code":"error.field.unknown"},{"field":"zeta","code":"error.field.unknown"}]`,
		},
		{
			name:        "old a and b names are unknown",
			contentType: binding.MIMEJSON,
			body:        `{"a":1,"b":2}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.required"},{"field":"rightOperand","code":"error.field.required"},{"field":"a","code":"error.field.unknown"},{"field":"b","code":"error.field.unknown"}]`,
		},
		{
			name:        "duplicate property: the last value wins",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"leftOperand":"x","rightOperand":2}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.invalid.type"}]`,
		},
		{
			name:        "every problem at once in a fixed order",
			contentType: binding.MIMEJSON,
			body:        `{"zzz":0,"precision":-1,"rightOperand":"x"}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.required"},{"field":"rightOperand","code":"error.field.invalid.type"},{"field":"precision","code":"error.field.out.of.range"},{"field":"zzz","code":"error.field.unknown"}]`,
		},

		// Unreadable bodies.
		{
			name:        "empty body",
			contentType: binding.MIMEJSON,
			body:        ``,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "malformed JSON",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "JSON array instead of an object",
			contentType: binding.MIMEJSON,
			body:        `[1,2]`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "JSON null instead of an object",
			contentType: binding.MIMEJSON,
			body:        `null`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "JSON number instead of an object",
			contentType: binding.MIMEJSON,
			body:        `3`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "second object after the first",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2}{}`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "stray closing brace after the object",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2}}`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "body over 1 MiB",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2,"pad":"` + strings.Repeat("x", 1<<20) + `"}`,
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantBody:    `{"code":"error.request.body.too.large"}`,
		},
		{
			name:        "plain text content type",
			contentType: "text/plain",
			body:        `{"leftOperand":1,"rightOperand":2}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantBody:    `{"code":"error.request.unsupported.media.type"}`,
		},
		{
			name:        "no content type",
			contentType: "",
			body:        `{"leftOperand":1,"rightOperand":2}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantBody:    `{"code":"error.request.unsupported.media.type"}`,
		},
	}

	router := newTestRouter(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rec := serveJSON(t, router, PathAdd, tt.contentType, tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("POST %s %s status = %d, want %d", PathAdd, truncate(tt.body), rec.Code, tt.wantStatus)
			}
			assertJSONEqual(t, tt.wantBody, rec.Body.Bytes())
			assertMatchesSpec(t, req, rec)
		})
	}
}

// TestSubtract pins the subtraction itself. Body validation is shared with
// every two-operand operation and covered in TestAdd; the few rows here prove
// the route uses it.
func TestSubtract(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "subtracts integers",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":5,"rightOperand":3}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":2}`,
		},
		{
			name:        "operand order matters",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":3,"rightOperand":5}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":-2}`,
		},
		{
			name:        "subtracting a negative adds",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-1.5,"rightOperand":-2.25}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.75}`,
		},
		{
			name:        "keeps float noise without precision",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0.3,"rightOperand":0.1}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.19999999999999998}`,
		},
		{
			name:        "rounds to precision",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0.3,"rightOperand":0.1,"precision":2}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.2}`,
		},
		{
			name:        "number minus itself is zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1.5,"rightOperand":1.5}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0}`,
		},
		{
			name:        "positive overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1.7e308,"rightOperand":-1.7e308}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "negative overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-1.7e308,"rightOperand":1.7e308}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "every field problem at once",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":"5","precision":16}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.invalid.type"},{"field":"rightOperand","code":"error.field.required"},{"field":"precision","code":"error.field.out.of.range"}]`,
		},
		{
			name:        "malformed JSON",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":5`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "plain text content type",
			contentType: "text/plain",
			body:        `{"leftOperand":5,"rightOperand":3}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantBody:    `{"code":"error.request.unsupported.media.type"}`,
		},
	}

	router := newTestRouter(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rec := serveJSON(t, router, PathSubtract, tt.contentType, tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("POST %s %s status = %d, want %d", PathSubtract, tt.body, rec.Code, tt.wantStatus)
			}
			assertJSONEqual(t, tt.wantBody, rec.Body.Bytes())
			assertMatchesSpec(t, req, rec)
		})
	}
}

// TestMultiply pins the multiplication itself and its float64 limits. Body
// validation is shared and covered in TestAdd; a few rows prove the route uses
// it.
func TestMultiply(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "multiplies integers",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":6,"rightOperand":7}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":42}`,
		},
		{
			name:        "negative times positive",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-2.5,"rightOperand":4}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":-10}`,
		},
		{
			name:        "keeps float noise without precision",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1.1,"rightOperand":1.1}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.2100000000000002}`,
		},
		{
			name:        "rounds to precision",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1.1,"rightOperand":1.1,"precision":2}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.21}`,
		},
		{
			name:        "integer product above 2^53 loses precision without error",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":94906267,"rightOperand":94906267}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":9007199515875288}`,
		},
		{
			name:        "largest product of equal powers of ten",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e154,"rightOperand":1e154}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1e308}`,
		},
		{
			name:        "positive overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e155,"rightOperand":1e155}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "negative overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-1e155,"rightOperand":1e155}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "subnormal product is kept",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e-160,"rightOperand":1e-160}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1e-320}`,
		},
		{
			name:        "product below the lower limit underflows to zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e-200,"rightOperand":1e-200}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0}`,
		},
		{
			name:        "every field problem at once",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e309,"rightOperand":true,"factor":2}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.out.of.range"},{"field":"rightOperand","code":"error.field.invalid.type"},{"field":"factor","code":"error.field.unknown"}]`,
		},
		{
			name:        "malformed JSON",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":6,"rightOperand":7`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "plain text content type",
			contentType: "text/plain",
			body:        `{"leftOperand":6,"rightOperand":7}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantBody:    `{"code":"error.request.unsupported.media.type"}`,
		},
	}

	router := newTestRouter(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rec := serveJSON(t, router, PathMultiply, tt.contentType, tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("POST %s %s status = %d, want %d", PathMultiply, tt.body, rec.Code, tt.wantStatus)
			}
			assertJSONEqual(t, tt.wantBody, rec.Body.Bytes())
			assertMatchesSpec(t, req, rec)
		})
	}
}

// TestDivide pins the division itself, division by zero and the float64
// limits. Body validation is shared and covered in TestAdd; a few rows prove
// the route uses it.
func TestDivide(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "exact quotient",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":10,"rightOperand":4}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":2.5}`,
		},
		{
			name:        "operand order matters",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":4,"rightOperand":10}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.4}`,
		},
		{
			name:        "repeating decimal",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":3}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.3333333333333333}`,
		},
		{
			name:        "rounds to precision",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":2,"rightOperand":3,"precision":2}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.67}`,
		},
		{
			name:        "zero dividend",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0,"rightOperand":7}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0}`,
		},
		{
			name:        "division by zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":0}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.division.by.zero"}`,
		},
		{
			name:        "division by negative zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-1,"rightOperand":-0}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.division.by.zero"}`,
		},
		{
			name:        "zero by zero is division by zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0,"rightOperand":0.0}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.division.by.zero"}`,
		},
		{
			name:        "divisor that underflows to zero is division by zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":1e-400}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.division.by.zero"}`,
		},
		{
			name:        "divisor just below the input cut-off reads as zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2.4e-324}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.division.by.zero"}`,
		},
		{
			name:        "divisor just above the input cut-off reads as the smallest float and overflows",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":2.5e-324}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "division by zero with precision",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":0,"precision":2}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.division.by.zero"}`,
		},
		{
			name:        "invalid divisor is a field error, not division by zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":"0"}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"rightOperand","code":"error.field.invalid.type"}]`,
		},
		{
			name:        "positive overflow from a tiny divisor",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":5e-324}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "negative overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-1e300,"rightOperand":1e-10}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "subnormal quotient is kept",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e-310,"rightOperand":10}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1e-311}`,
		},
		{
			name:        "quotient below the lower limit underflows to zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e-300,"rightOperand":1e300}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0}`,
		},
		{
			name:        "every field problem at once",
			contentType: binding.MIMEJSON,
			body:        `{"rightOperand":null,"precision":"2"}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.required"},{"field":"rightOperand","code":"error.field.required"},{"field":"precision","code":"error.field.invalid.type"}]`,
		},
		{
			name:        "malformed JSON",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1,"rightOperand":0`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "plain text content type",
			contentType: "text/plain",
			body:        `{"leftOperand":10,"rightOperand":4}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantBody:    `{"code":"error.request.unsupported.media.type"}`,
		},
	}

	router := newTestRouter(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rec := serveJSON(t, router, PathDivide, tt.contentType, tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("POST %s %s status = %d, want %d", PathDivide, tt.body, rec.Code, tt.wantStatus)
			}
			assertJSONEqual(t, tt.wantBody, rec.Body.Bytes())
			assertMatchesSpec(t, req, rec)
		})
	}
}

// TestPower pins exponentiation, its own error (zero to a negative power), the
// undefined and exactness cases, and the float64 limits. Body validation is
// shared and covered in TestAdd; a few rows prove the route uses it.
func TestPower(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "integer power",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":2,"rightOperand":10}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1024}`,
		},
		{
			name:        "fractional exponent",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":2,"rightOperand":0.5}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.4142135623730951}`,
		},
		{
			name:        "rounds to precision",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":2,"rightOperand":0.5,"precision":3}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.414}`,
		},
		{
			name:        "negative base to an odd exponent",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-2,"rightOperand":3}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":-8}`,
		},
		{
			name:        "zero to the zero is one",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0,"rightOperand":0}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1}`,
		},
		{
			name:        "largest power of ten is exact",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":10,"rightOperand":308}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1e308}`,
		},
		{
			name:        "zero to a negative power",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":0,"rightOperand":-1}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.zero.negative.power"}`,
		},
		{
			name:        "negative zero to a negative even power",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-0,"rightOperand":-2}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.zero.negative.power"}`,
		},
		{
			name:        "base that underflows to zero, negative power",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e-400,"rightOperand":-1}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.zero.negative.power"}`,
		},
		{
			name:        "negative base with a fractional exponent is undefined",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-8,"rightOperand":0.5}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.undefined"}`,
		},
		{
			name:        "positive overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":10,"rightOperand":309}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "negative overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-10,"rightOperand":309}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "smallest power of two in range",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":2,"rightOperand":-1074}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":5e-324}`,
		},
		{
			name:        "power below the lower limit underflows to zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":10,"rightOperand":-400}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0}`,
		},
		{
			name:        "every field problem at once",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":[2],"rightOperand":1e999,"precision":1.5}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.invalid.type"},{"field":"rightOperand","code":"error.field.out.of.range"},{"field":"precision","code":"error.field.invalid.type"}]`,
		},
		{
			name:        "malformed JSON",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":2,"rightOperand":10`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "plain text content type",
			contentType: "text/plain",
			body:        `{"leftOperand":2,"rightOperand":10}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantBody:    `{"code":"error.request.unsupported.media.type"}`,
		},
	}

	router := newTestRouter(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rec := serveJSON(t, router, PathPower, tt.contentType, tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("POST %s %s status = %d, want %d", PathPower, tt.body, rec.Code, tt.wantStatus)
			}
			assertJSONEqual(t, tt.wantBody, rec.Body.Bytes())
			assertMatchesSpec(t, req, rec)
		})
	}
}

// TestPercentage pins "leftOperand percent of rightOperand", its exactness and
// the float64 limits. Body validation is shared and covered in TestAdd; a few
// rows prove the route uses it.
func TestPercentage(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "whole percent",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":10,"rightOperand":50}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":5}`,
		},
		{
			name:        "swapping the operands gives the same result",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":50,"rightOperand":10}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":5}`,
		},
		{
			name:        "negative percent",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-20,"rightOperand":50}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":-10}`,
		},
		{
			name:        "two-decimal inputs give the expected decimal",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":15,"rightOperand":19.99}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":2.9985}`,
		},
		{
			name:        "seven percent of a hundred is exactly seven",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":7,"rightOperand":100}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":7}`,
		},
		{
			name:        "rounds to precision",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":15,"rightOperand":19.99,"precision":2}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":3}`,
		},
		{
			name:        "large percent whose product alone would overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e307,"rightOperand":50}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":5e306}`,
		},
		{
			name:        "positive overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":200,"rightOperand":1.7976931348623157e308}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "negative overflow",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":-1e308,"rightOperand":1e308}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.result.overflow"}`,
		},
		{
			name:        "smallest float percent of a hundred is kept",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":5e-324,"rightOperand":100}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":5e-324}`,
		},
		{
			name:        "result below the lower limit underflows to zero",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":1e-300,"rightOperand":1e-300}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0}`,
		},
		{
			name:        "every field problem at once",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":"10%","percent":10}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"leftOperand","code":"error.field.invalid.type"},{"field":"rightOperand","code":"error.field.required"},{"field":"percent","code":"error.field.unknown"}]`,
		},
		{
			name:        "malformed JSON",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":10,"rightOperand":50`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "plain text content type",
			contentType: "text/plain",
			body:        `{"leftOperand":10,"rightOperand":50}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantBody:    `{"code":"error.request.unsupported.media.type"}`,
		},
	}

	router := newTestRouter(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rec := serveJSON(t, router, PathPercentage, tt.contentType, tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("POST %s %s status = %d, want %d", PathPercentage, tt.body, rec.Code, tt.wantStatus)
			}
			assertJSONEqual(t, tt.wantBody, rec.Body.Bytes())
			assertMatchesSpec(t, req, rec)
		})
	}
}

// TestSqrt pins the square root, its one-operand body ({"operand",
// "precision"?}) and its decimal operands. The body has its own decoder, so
// unlike the two-operand operations its validation is tested here in full.
func TestSqrt(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantBody    string
	}{
		// Success.
		{
			name:        "perfect square",
			contentType: binding.MIMEJSON,
			body:        `{"operand":16}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":4}`,
		},
		{
			name:        "irrational root",
			contentType: binding.MIMEJSON,
			body:        `{"operand":2}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.4142135623730951}`,
		},
		{
			name:        "rounds to precision",
			contentType: binding.MIMEJSON,
			body:        `{"operand":2,"precision":3}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.414}`,
		},
		{
			name:        "zero",
			contentType: binding.MIMEJSON,
			body:        `{"operand":0}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0}`,
		},
		{
			name:        "decimal operand",
			contentType: binding.MIMEJSON,
			body:        `{"operand":1.5}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.224744871391589}`,
		},
		{
			name:        "decimal with an exact root",
			contentType: binding.MIMEJSON,
			body:        `{"operand":2.25}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.5}`,
		},
		{
			name:        "decimal below one has a larger root",
			contentType: binding.MIMEJSON,
			body:        `{"operand":0.01}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0.1}`,
		},
		{
			name:        "decimal rounded to precision",
			contentType: binding.MIMEJSON,
			body:        `{"operand":1.5,"precision":4}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.2247}`,
		},
		{
			name:        "smallest positive float stays in range",
			contentType: binding.MIMEJSON,
			body:        `{"operand":5e-324}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":2.2227587494850775e-162}`,
		},
		{
			name:        "whole number written with a decimal point",
			contentType: binding.MIMEJSON,
			body:        `{"operand":9.0}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":3}`,
		},
		{
			name:        "whole number in exponent notation",
			contentType: binding.MIMEJSON,
			body:        `{"operand":1e20}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1e10}`,
		},
		{
			name:        "largest float stays in range",
			contentType: binding.MIMEJSON,
			body:        `{"operand":1.7976931348623157e308}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.3407807929942596e154}`,
		},
		{
			name:        "operand below the float64 range underflows to zero",
			contentType: binding.MIMEJSON,
			body:        `{"operand":1e-400}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":0}`,
		},
		{
			name:        "null precision means no rounding",
			contentType: binding.MIMEJSON,
			body:        `{"operand":2,"precision":null}`,
			wantStatus:  http.StatusOK,
			wantBody:    `{"result":1.4142135623730951}`,
		},

		// Math error.
		{
			name:        "negative whole number",
			contentType: binding.MIMEJSON,
			body:        `{"operand":-4}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.negative.square.root"}`,
		},
		{
			name:        "negative decimal",
			contentType: binding.MIMEJSON,
			body:        `{"operand":-2.5}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.negative.square.root"}`,
		},
		{
			name:        "tiny negative decimal",
			contentType: binding.MIMEJSON,
			body:        `{"operand":-1e-320}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.negative.square.root"}`,
		},
		{
			name:        "negative whole number with precision",
			contentType: binding.MIMEJSON,
			body:        `{"operand":-1,"precision":2}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `{"code":"error.calculation.negative.square.root"}`,
		},

		// Field errors.
		{
			name:        "missing operand",
			contentType: binding.MIMEJSON,
			body:        `{}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"operand","code":"error.field.required"}]`,
		},
		{
			name:        "null operand",
			contentType: binding.MIMEJSON,
			body:        `{"operand":null}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"operand","code":"error.field.required"}]`,
		},
		{
			name:        "numeric string",
			contentType: binding.MIMEJSON,
			body:        `{"operand":"16"}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"operand","code":"error.field.invalid.type"}]`,
		},
		{
			name:        "operand beyond float64 range",
			contentType: binding.MIMEJSON,
			body:        `{"operand":1e309}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"operand","code":"error.field.out.of.range"}]`,
		},
		{
			name:        "two-operand fields are unknown here",
			contentType: binding.MIMEJSON,
			body:        `{"leftOperand":16}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"operand","code":"error.field.required"},{"field":"leftOperand","code":"error.field.unknown"}]`,
		},
		{
			name:        "every problem at once in a fixed order",
			contentType: binding.MIMEJSON,
			body:        `{"zeta":1,"precision":16,"operand":"2.5","alpha":1}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantBody:    `[{"field":"operand","code":"error.field.invalid.type"},{"field":"precision","code":"error.field.out.of.range"},{"field":"alpha","code":"error.field.unknown"},{"field":"zeta","code":"error.field.unknown"}]`,
		},

		// Unreadable bodies.
		{
			name:        "malformed JSON",
			contentType: binding.MIMEJSON,
			body:        `{"operand":16`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "JSON number instead of an object",
			contentType: binding.MIMEJSON,
			body:        `16`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    `{"code":"error.request.invalid.json"}`,
		},
		{
			name:        "plain text content type",
			contentType: "text/plain",
			body:        `{"operand":16}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantBody:    `{"code":"error.request.unsupported.media.type"}`,
		},
	}

	router := newTestRouter(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rec := serveJSON(t, router, PathSqrt, tt.contentType, tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("POST %s %s status = %d, want %d", PathSqrt, tt.body, rec.Code, tt.wantStatus)
			}
			assertJSONEqual(t, tt.wantBody, rec.Body.Bytes())
			assertMatchesSpec(t, req, rec)
		})
	}
}

// TestOperationsNeverReturnNegativeZero checks the raw body: a semantic JSON
// comparison treats -0 and 0 as equal.
func TestOperationsNeverReturnNegativeZero(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "add negative zero operands",
			path: PathAdd,
			body: `{"leftOperand":-0,"rightOperand":-0}`,
		},
		{
			name: "add negative underflow",
			path: PathAdd,
			body: `{"leftOperand":-1e-400,"rightOperand":-0}`,
		},
		{
			name: "add rounding a small negative number",
			path: PathAdd,
			body: `{"leftOperand":-0.001,"rightOperand":0,"precision":2}`,
		},
		{
			name: "subtract zero from negative zero",
			path: PathSubtract,
			body: `{"leftOperand":-0,"rightOperand":0}`,
		},
		{
			name: "subtract rounding a small negative number",
			path: PathSubtract,
			body: `{"leftOperand":0,"rightOperand":0.001,"precision":2}`,
		},
		{
			name: "multiply zero by a negative number",
			path: PathMultiply,
			body: `{"leftOperand":0,"rightOperand":-5}`,
		},
		{
			name: "multiply negative underflow",
			path: PathMultiply,
			body: `{"leftOperand":-1e-200,"rightOperand":1e-200}`,
		},
		{
			name: "multiply rounding a small negative product",
			path: PathMultiply,
			body: `{"leftOperand":-0.01,"rightOperand":0.1,"precision":2}`,
		},
		{
			name: "divide zero by a negative number",
			path: PathDivide,
			body: `{"leftOperand":0,"rightOperand":-5}`,
		},
		{
			name: "divide negative underflow",
			path: PathDivide,
			body: `{"leftOperand":-1e-300,"rightOperand":1e300}`,
		},
		{
			name: "divide rounding a small negative quotient",
			path: PathDivide,
			body: `{"leftOperand":-1,"rightOperand":1000,"precision":2}`,
		},
		{
			name: "power of negative zero to an odd exponent",
			path: PathPower,
			body: `{"leftOperand":-0,"rightOperand":3}`,
		},
		{
			name: "power negative underflow",
			path: PathPower,
			body: `{"leftOperand":-10,"rightOperand":-401}`,
		},
		{
			name: "power rounding a small negative result",
			path: PathPower,
			body: `{"leftOperand":-0.1,"rightOperand":3,"precision":2}`,
		},
		{
			name: "percentage of a negative number at zero percent",
			path: PathPercentage,
			body: `{"leftOperand":0,"rightOperand":-5}`,
		},
		{
			name: "percentage negative underflow",
			path: PathPercentage,
			body: `{"leftOperand":-1e-300,"rightOperand":1e-300}`,
		},
		{
			name: "percentage rounding a small negative result",
			path: PathPercentage,
			body: `{"leftOperand":-1,"rightOperand":0.1,"precision":2}`,
		},
		{
			name: "sqrt of negative zero",
			path: PathSqrt,
			body: `{"operand":-0}`,
		},
		{
			name: "sqrt of an operand that underflows to negative zero",
			path: PathSqrt,
			body: `{"operand":-1e-400}`,
		},
	}

	router := newTestRouter(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, rec := serveJSON(t, router, tt.path, binding.MIMEJSON, tt.body)

			if got, want := rec.Body.String(), `{"result":0}`; got != want {
				t.Errorf("POST %s %s body = %#v, want %#v", tt.path, tt.body, got, want)
			}
		})
	}
}

// fakeCalculator fails every operation with err, to reach responses the real
// calculator can't produce.
type fakeCalculator struct {
	err error
}

func (f fakeCalculator) Add(float64, float64, service.Precision) (float64, error) {
	return 0, f.err
}

func (f fakeCalculator) Subtract(float64, float64, service.Precision) (float64, error) {
	return 0, f.err
}

func (f fakeCalculator) Multiply(float64, float64, service.Precision) (float64, error) {
	return 0, f.err
}

func (f fakeCalculator) Divide(float64, float64, service.Precision) (float64, error) {
	return 0, f.err
}

func (f fakeCalculator) Power(float64, float64, service.Precision) (float64, error) {
	return 0, f.err
}

func (f fakeCalculator) Percentage(float64, float64, service.Precision) (float64, error) {
	return 0, f.err
}

func (f fakeCalculator) Sqrt(float64, service.Precision) (float64, error) {
	return 0, f.err
}

var errDiskOnFire = errors.New("disk on fire")

// TestOperationServiceErrors checks how every operation route answers the
// errors its Calculator method returns.
func TestOperationServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "wrapped expected failure keeps its code",
			err:        fmt.Errorf("operation: %w", apperror.ErrResultUndefined),
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   `{"code":"error.calculation.result.undefined"}`,
		},
		{
			name:       "internal failure reveals only the internal code",
			err:        apperror.Internal(errDiskOnFire),
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"code":"error.internal"}`,
		},
		{
			name:       "validation error without fields is internal",
			err:        apperror.Validation(),
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"code":"error.internal"}`,
		},
		{
			name:       "plain error is internal",
			err:        errDiskOnFire,
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"code":"error.internal"}`,
		},
	}

	operations := []struct {
		name string
		path string
		body string
	}{
		{name: "add", path: PathAdd, body: `{"leftOperand":1,"rightOperand":2}`},
		{name: "subtract", path: PathSubtract, body: `{"leftOperand":1,"rightOperand":2}`},
		{name: "multiply", path: PathMultiply, body: `{"leftOperand":1,"rightOperand":2}`},
		{name: "divide", path: PathDivide, body: `{"leftOperand":1,"rightOperand":2}`},
		{name: "power", path: PathPower, body: `{"leftOperand":1,"rightOperand":2}`},
		{name: "percentage", path: PathPercentage, body: `{"leftOperand":1,"rightOperand":2}`},
		{name: "sqrt", path: PathSqrt, body: `{"operand":4}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newRouterWith(t, fakeCalculator{err: tt.err})
			for _, op := range operations {
				t.Run(op.name, func(t *testing.T) {
					req, rec := serveJSON(t, router, op.path, binding.MIMEJSON, op.body)

					if rec.Code != tt.wantStatus {
						t.Errorf("POST %s with calculator error %v: status = %d, want %d", op.path, tt.err, rec.Code, tt.wantStatus)
					}
					assertJSONEqual(t, tt.wantBody, rec.Body.Bytes())
					assertMatchesSpec(t, req, rec)
				})
			}
		})
	}
}

// truncate keeps failure messages readable for very large bodies.
func truncate(body string) string {
	maxLen := 80
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "..."
}
