package handler

import (
	"net/http"
	"regexp"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
)

// docRoutes serve the documentation itself; they are not part of the API
// contract, so they are not described in api/openapi.yaml.
var docRoutes = []string{
	http.MethodGet + " " + pathSwaggerUI,
	http.MethodGet + " " + pathSwaggerSpec,
}

// ginParam matches gin path parameters (":id") so they can be compared with
// OpenAPI templates ("{id}").
var ginParam = regexp.MustCompile(`:([A-Za-z0-9_]+)`)

// TestRoutesMatchSpec keeps the router and api/openapi.yaml in sync: every
// registered API route must be documented, and every documented operation must
// be registered.
func TestRoutesMatchSpec(t *testing.T) {
	router := newTestRouter(t)
	engine, ok := router.(*gin.Engine)
	if !ok {
		t.Fatalf("NewRouter() = %T, want *gin.Engine", router)
	}

	var registered []string
	for _, r := range engine.Routes() {
		route := r.Method + " " + ginParam.ReplaceAllString(r.Path, "{$1}")
		if !slices.Contains(docRoutes, route) {
			registered = append(registered, route)
		}
	}

	var documented []string
	for path, item := range loadSpec(t).Paths.Map() {
		for method := range item.Operations() {
			documented = append(documented, method+" "+path)
		}
	}

	slices.Sort(registered)
	slices.Sort(documented)
	if diff := cmp.Diff(documented, registered); diff != "" {
		t.Errorf("router and api/openapi.yaml disagree (-documented +registered):\n%s", diff)
	}
}

func TestRouterFallbacks(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/missing",
			wantStatus: http.StatusNotFound,
			wantBody:   `{"code":"error.request.not.found"}`,
		},
		{
			name:       "wrong method on a known path",
			method:     http.MethodPost,
			path:       "/health",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `{"code":"error.request.method.not.allowed"}`,
		},
		{
			name:       "GET on an operation",
			method:     http.MethodGet,
			path:       "/api/v1/add",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `{"code":"error.request.method.not.allowed"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, rec := serve(t, tt.method, tt.path)

			if rec.Code != tt.wantStatus {
				t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("%s %s Content-Type = %q, want JSON", tt.method, tt.path, ct)
			}
			assertJSONEqual(t, tt.wantBody, rec.Body.Bytes())
			assertMatchesSchema(t, "ErrorResponse", rec.Body.Bytes())
		})
	}
}
