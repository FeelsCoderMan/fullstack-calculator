package handler

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestHealth(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		wantStatus      int
		wantContentType string
		wantBody        string
	}{
		{
			name:            "GET reports ok",
			method:          http.MethodGet,
			wantStatus:      http.StatusOK,
			wantContentType: "application/json; charset=utf-8",
			wantBody:        `{"status":"ok"}`,
		},
		{
			// The recorder keeps the body; the real server drops it on the wire for HEAD.
			name:            "HEAD is routed to the GET handler",
			method:          http.MethodHead,
			wantStatus:      http.StatusOK,
			wantContentType: "application/json; charset=utf-8",
			wantBody:        `{"status":"ok"}`,
		},
		{
			name:            "POST is not allowed",
			method:          http.MethodPost,
			wantStatus:      http.StatusMethodNotAllowed,
			wantContentType: "text/plain",
			wantBody:        "405 method not allowed",
		},
	}

	router := NewRouter(discardLogger())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tt.method, "/health", http.NoBody))

			if rec.Code != tt.wantStatus {
				t.Errorf("%s /health status = %d, want %d", tt.method, rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != tt.wantContentType {
				t.Errorf("%s /health Content-Type = %q, want %q", tt.method, ct, tt.wantContentType)
			}
			if got := rec.Body.String(); got != tt.wantBody {
				t.Errorf("%s /health body = %q, want %q", tt.method, got, tt.wantBody)
			}
		})
	}
}
