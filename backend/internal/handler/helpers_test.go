package handler

import (
	"context"
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

	"fullstack-calculator/backend/api"
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

// serve sends a request through the real router and returns the recorded response.
func serve(t *testing.T, method, path string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, http.NoBody)
	rec := httptest.NewRecorder()
	NewRouter(discardLogger()).ServeHTTP(rec, req)
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
