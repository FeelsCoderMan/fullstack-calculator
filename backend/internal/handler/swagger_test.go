package handler

import (
	"net/http"
	"testing"

	"fullstack-calculator/backend/api"
)

func TestSwagger(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		wantStatus      int
		wantContentType string
		wantBody        []byte
		wantLocation    string
	}{
		{
			name:            "UI page",
			path:            "/swagger/",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html; charset=utf-8",
			wantBody:        api.SwaggerUI,
		},
		{
			name:            "raw spec",
			path:            "/swagger/openapi.yaml",
			wantStatus:      http.StatusOK,
			wantContentType: "application/yaml",
			wantBody:        api.Spec,
		},
		{
			name:         "missing trailing slash redirects to the UI",
			path:         "/swagger",
			wantStatus:   http.StatusMovedPermanently,
			wantLocation: "/swagger/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, rec := serve(t, http.MethodGet, tt.path)

			if rec.Code != tt.wantStatus {
				t.Errorf("GET %s status = %d, want %d", tt.path, rec.Code, tt.wantStatus)
			}
			if tt.wantContentType != "" {
				if ct := rec.Header().Get("Content-Type"); ct != tt.wantContentType {
					t.Errorf("GET %s Content-Type = %q, want %q", tt.path, ct, tt.wantContentType)
				}
			}
			if tt.wantBody != nil && rec.Body.String() != string(tt.wantBody) {
				t.Errorf("GET %s body differs from the embedded file (%d bytes, want %d)",
					tt.path, rec.Body.Len(), len(tt.wantBody))
			}
			if loc := rec.Header().Get("Location"); loc != tt.wantLocation {
				t.Errorf("GET %s Location = %q, want %q", tt.path, loc, tt.wantLocation)
			}
		})
	}
}
