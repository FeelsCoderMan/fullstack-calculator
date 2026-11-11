package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/google/go-cmp/cmp"

	"fullstack-calculator/backend/api"
	"fullstack-calculator/backend/internal/service"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// spec parses and validates api/openapi.yaml once per test binary.
var spec = sync.OnceValues(func() (*openapi3.T, error) {
	doc, err := openapi3.NewLoader().LoadFromData(api.Spec)
	if err != nil {
		return nil, err
	}
	if err := doc.Validate(context.Background()); err != nil {
		return nil, err
	}
	return doc, nil
})

func loadSpec(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := spec()
	if err != nil {
		t.Fatalf("api/openapi.yaml is invalid: %v", err)
	}
	return doc
}

// newTestRouter is the real router with the real calculator.
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	return newRouterWith(t, service.NewCalculator())
}

// newRouterWith is the real router serving the operations with calc.
func newRouterWith(t *testing.T, calc Calculator) http.Handler {
	t.Helper()
	router, err := NewRouter(discardLogger(), calc)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	return router
}

// serve sends a bodiless request through the real router and returns the
// recorded response.
func serve(t *testing.T, method, path string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, http.NoBody)
	rec := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(rec, req)
	return req, rec
}

// serveJSON POSTs body with the given Content-Type through router and returns
// the recorded response.
func serveJSON(t *testing.T, router http.Handler, path, contentType, body string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return req, rec
}

// assertMatchesSpec fails unless the response is exactly what api/openapi.yaml
// documents for that operation: a declared status code and a body that matches
// its schema (or no body when none is declared). This is what keeps the spec
// the frontend relies on honest.
func assertMatchesSpec(t *testing.T, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()
	doc := loadSpec(t)
	router, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatalf("build spec router: %v", err)
	}
	route, pathParams, err := router.FindRoute(req)
	if err != nil {
		t.Fatalf("%s %s is not documented in api/openapi.yaml: %v", req.Method, req.URL.Path, err)
	}
	documented := route.Operation.Responses.Status(rec.Code)
	if documented == nil {
		t.Fatalf("%s %s returned %d, which api/openapi.yaml does not document", req.Method, req.URL.Path, rec.Code)
	}
	if documented.Value.Content == nil && rec.Body.Len() > 0 {
		t.Fatalf("%s %s returned a body, but api/openapi.yaml documents none for %d: %q",
			req.Method, req.URL.Path, rec.Code, rec.Body.String())
	}
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		},
		Status: rec.Code,
		Header: rec.Header(),
		Body:   io.NopCloser(strings.NewReader(rec.Body.String())),
	}
	if err := openapi3filter.ValidateResponse(t.Context(), input); err != nil {
		t.Fatalf("%s %s response does not match api/openapi.yaml: %v", req.Method, req.URL.Path, err)
	}
}

// assertMatchesSchema fails unless body is valid JSON matching the named
// schema in api/openapi.yaml components (for responses that are not tied to a
// documented operation, such as the 404/405 fallbacks).
func assertMatchesSchema(t *testing.T, schema string, body []byte) {
	t.Helper()
	ref, ok := loadSpec(t).Components.Schemas[schema]
	if !ok {
		t.Fatalf("api/openapi.yaml has no schema %q", schema)
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("body is not JSON: %q: %v", body, err)
	}
	if err := ref.Value.VisitJSON(v); err != nil {
		t.Fatalf("body %s does not match schema %s: %v", body, schema, err)
	}
}

// assertJSONEqual compares JSON semantically (key order and whitespace don't
// matter; array order does).
func assertJSONEqual(t *testing.T, want string, got []byte) {
	t.Helper()
	var w, g any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("bad want JSON %q: %v", want, err)
	}
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("body is not JSON: %q: %v", got, err)
	}
	if diff := cmp.Diff(w, g); diff != "" {
		t.Errorf("body mismatch (-want +got):\n%s", diff)
	}
}
