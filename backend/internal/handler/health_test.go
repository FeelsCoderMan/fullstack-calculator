package handler

import (
	"net/http"
	"testing"
)

func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "GET answers with a status code only",
			method:     http.MethodGet,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "HEAD answers like GET",
			method:     http.MethodHead,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rec := serve(t, tt.method, "/health")

			if rec.Code != tt.wantStatus {
				t.Errorf("%s /health status = %d, want %d", tt.method, rec.Code, tt.wantStatus)
			}
			if got := rec.Body.String(); got != "" {
				t.Errorf("%s /health body = %q, want no body", tt.method, got)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "" {
				t.Errorf("%s /health Content-Type = %q, want none", tt.method, ct)
			}
			assertMatchesSpec(t, req, rec)
		})
	}
}
