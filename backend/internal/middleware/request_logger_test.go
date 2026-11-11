package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"

	"fullstack-calculator/backend/internal/apperror"
)

// debugPath is the path RequestLogger is told to log at debug level when it
// succeeds, as NewRouter does with /health.
const debugPath = "/health"

// errorBody is the {"code": ...} body writeTestError answers with.
type errorBody struct {
	Code apperror.Code `json:"code"`
}

// writeTestError stands in for the handler's writeError: it records the
// failure and answers 500 with the code.
func writeTestError(c *gin.Context, err error) {
	appErr := apperror.From(err)
	RecordFailure(c, Failure{Err: err, AppErr: appErr, Code: appErr.Code})
	c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody{Code: appErr.Code})
}

// records decodes every JSON line written to buf.
func records(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line is not JSON: %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

// onlyRecord fails unless exactly one record was written, and returns it.
func onlyRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	recs := records(t, buf)
	if len(recs) != 1 {
		t.Fatalf("got %d log records, want exactly 1:\n%s", len(recs), buf.String())
	}
	return recs[0]
}

// serveWith runs one request through RequestLogger (with writeTestError and
// debugPath) and handler h, capturing logs at debug level.
func serveWith(t *testing.T, method, path string, h gin.HandlerFunc, header http.Header) (*httptest.ResponseRecorder, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	accessLog := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(RequestLogger(accessLog, writeTestError, debugPath))
	r.Handle(method, path, h)

	req := httptest.NewRequestWithContext(t.Context(), method, path, http.NoBody)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec, &buf
}

func TestRequestLoggerWritesOneRecord(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		handler    gin.HandlerFunc
		wantStatus int
		wantBody   string
		wantLevel  string
		wantCode   any // nil = attribute absent
		wantError  any
		wantStack  bool
	}{
		{
			name:       "success is info",
			path:       "/api/v1/add",
			handler:    func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"result": 3}) },
			wantStatus: http.StatusOK,
			wantBody:   `{"result":3}`,
			wantLevel:  "INFO",
		},
		{
			name:       "success on a debug path is debug",
			path:       debugPath,
			handler:    func(c *gin.Context) { c.Status(http.StatusNoContent) },
			wantStatus: http.StatusNoContent,
			wantLevel:  "DEBUG",
		},
		{
			name:       "server failure on a debug path is still an error",
			path:       debugPath,
			handler:    func(c *gin.Context) { c.Status(http.StatusServiceUnavailable) },
			wantStatus: http.StatusServiceUnavailable,
			wantLevel:  "ERROR",
		},
		{
			name: "recorded expected failure is info with code and error, no stack",
			path: "/api/v1/divide",
			handler: func(c *gin.Context) {
				err := fmt.Errorf("divide 1 by 0: %w", apperror.ErrDivisionByZero)
				RecordFailure(c, Failure{Err: err, AppErr: apperror.From(err), Code: apperror.CodeDivisionByZero})
				c.JSON(http.StatusUnprocessableEntity, errorBody{Code: apperror.CodeDivisionByZero})
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   `{"code":"error.calculation.division.by.zero"}`,
			wantLevel:  "INFO",
			wantCode:   "error.calculation.division.by.zero",
			wantError:  "divide 1 by 0: error.calculation.division.by.zero",
		},
		{
			name:       "panic is answered by the error writer and logged once",
			path:       "/api/v1/sqrt",
			handler:    func(*gin.Context) { panic("boom") },
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"code":"error.internal"}`,
			wantLevel:  "ERROR",
			wantCode:   "error.internal",
			wantError:  "error.internal: panic: boom",
			wantStack:  true,
		},
		{
			name: "panic after the response was written keeps the response",
			path: "/api/v1/sqrt",
			handler: func(c *gin.Context) {
				c.String(http.StatusOK, "partial")
				panic("late boom")
			},
			wantStatus: http.StatusOK,
			wantLevel:  "ERROR",
			wantCode:   "error.internal",
			wantError:  "error.internal: panic: late boom",
			wantStack:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, buf := serveWith(t, http.MethodPost, tt.path, tt.handler, nil)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" {
				var want, got any
				if err := json.Unmarshal([]byte(tt.wantBody), &want); err != nil {
					t.Fatalf("bad want JSON %q: %v", tt.wantBody, err)
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("body is not JSON: %q: %v", rec.Body.String(), err)
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("body mismatch (-want +got):\n%s", diff)
				}
			}

			got := onlyRecord(t, buf)
			want := map[string]any{
				"level":  tt.wantLevel,
				"msg":    "request completed",
				"method": "POST",
				"path":   tt.path,
				"status": float64(tt.wantStatus),
				"code":   tt.wantCode,
				"error":  tt.wantError,
			}
			for key, w := range want {
				if got[key] != w {
					t.Errorf("record[%q] = %#v, want %#v", key, got[key], w)
				}
			}
			if _, ok := got["duration_ms"].(float64); !ok {
				t.Errorf("record[duration_ms] = %#v, want a number", got["duration_ms"])
			}
			if got["request_id"] != rec.Header().Get(HeaderRequestID) {
				t.Errorf("record[request_id] = %#v, want the X-Request-ID response header %q",
					got["request_id"], rec.Header().Get(HeaderRequestID))
			}
			stack, hasStack := got["stack"].([]any)
			if hasStack != tt.wantStack || (tt.wantStack && len(stack) == 0) {
				t.Errorf("record[stack] = %#v, want present = %v", got["stack"], tt.wantStack)
			}
		})
	}
}

func TestRequestLoggerRepanicsAbortHandler(t *testing.T) {
	defer func() {
		if v := recover(); v != http.ErrAbortHandler { //nolint:errorlint // identity check on the recovered sentinel is intended
			t.Errorf("recovered %v, want http.ErrAbortHandler to keep propagating", v)
		}
	}()
	_, buf := serveWith(t, http.MethodGet, "/x", func(*gin.Context) { panic(http.ErrAbortHandler) }, nil)
	t.Errorf("serve returned normally; logs:\n%s", buf.String())
}

func TestRequestID(t *testing.T) {
	tests := []struct {
		name     string
		incoming string
		wantSame bool
	}{
		{
			name:     "well-formed incoming id is reused",
			incoming: "client-abc-123",
			wantSame: true,
		},
		{
			name:     "missing id is generated",
			incoming: "",
			wantSame: false,
		},
		{
			name:     "id with odd characters is replaced",
			incoming: "abc\"},{\"level\":\"ERROR",
			wantSame: false,
		},
		{
			name:     "id longer than 64 characters is replaced",
			incoming: strings.Repeat("a", 65),
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := http.Header{}
			if tt.incoming != "" {
				header.Set(HeaderRequestID, tt.incoming)
			}
			rec, buf := serveWith(t, http.MethodGet, "/x", func(c *gin.Context) { c.Status(http.StatusOK) }, header)

			got := rec.Header().Get(HeaderRequestID)
			if (got == tt.incoming) != tt.wantSame {
				t.Errorf("X-Request-ID = %q for incoming %q, want reused = %v", got, tt.incoming, tt.wantSame)
			}
			if !validRequestID.MatchString(got) {
				t.Errorf("X-Request-ID = %q does not match %s", got, requestIDPattern)
			}
			if rec := onlyRecord(t, buf); rec["request_id"] != got {
				t.Errorf("record[request_id] = %#v, want %q", rec["request_id"], got)
			}
		})
	}
}
