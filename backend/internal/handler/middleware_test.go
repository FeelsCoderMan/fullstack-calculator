package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func captureLogger(level slog.Level) (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: level})), &buf
}

func decodeRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log is not a single JSON record: %q: %v", buf.String(), err)
	}
	return rec
}

// testEngine returns a bare gin engine with AccessLog installed, in release
// mode so gin prints no debug output.
func testEngine(accessLog *slog.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(AccessLog(accessLog))
	return r
}

func TestAccessLogRecordsRequest(t *testing.T) {
	tests := []struct {
		name       string
		handler    gin.HandlerFunc
		wantStatus float64
	}{
		{
			name:       "explicit status",
			handler:    func(c *gin.Context) { c.Status(http.StatusTeapot) },
			wantStatus: http.StatusTeapot,
		},
		{
			name:       "implicit 200 on write",
			handler:    func(c *gin.Context) { _, _ = c.Writer.WriteString("hi") },
			wantStatus: http.StatusOK,
		},
		{
			name: "status is fixed once the body is written",
			handler: func(c *gin.Context) {
				c.String(http.StatusAccepted, "done")
				c.Status(http.StatusInternalServerError)
			},
			wantStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accessLog, buf := captureLogger(slog.LevelInfo)
			r := testEngine(accessLog)
			r.POST("/api/v1/add", tt.handler)

			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/add", http.NoBody))

			rec := decodeRecord(t, buf)
			if rec["msg"] != "request completed" || rec["level"] != "INFO" ||
				rec["method"] != "POST" || rec["path"] != "/api/v1/add" || rec["status"] != tt.wantStatus {
				t.Errorf("record = %v, want INFO request completed for POST /api/v1/add with status %v", rec, tt.wantStatus)
			}
			if d, ok := rec["duration_ms"].(float64); !ok || d < 0 {
				t.Errorf("duration_ms = %v, want non-negative number", rec["duration_ms"])
			}
		})
	}
}

func TestAccessLogLevel(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus float64
		wantLevel  string
	}{
		{
			name:       "successful health check is debug",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
			wantLevel:  "DEBUG",
		},
		{
			name:       "failed health check stays at info",
			method:     http.MethodPost,
			path:       "/health",
			wantStatus: http.StatusMethodNotAllowed,
			wantLevel:  "INFO",
		},
		{
			name:       "other requests are info",
			method:     http.MethodGet,
			path:       "/missing",
			wantStatus: http.StatusNotFound,
			wantLevel:  "INFO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accessLog, buf := captureLogger(slog.LevelDebug)

			NewRouter(accessLog).ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, http.NoBody))

			rec := decodeRecord(t, buf)
			if rec["status"] != tt.wantStatus {
				t.Errorf("%s %s logged status = %v, want %v", tt.method, tt.path, rec["status"], tt.wantStatus)
			}
			if rec["level"] != tt.wantLevel {
				t.Errorf("%s %s logged level = %v, want %v", tt.method, tt.path, rec["level"], tt.wantLevel)
			}
		})
	}
}
