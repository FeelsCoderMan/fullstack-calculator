package logger

import (
	"errors"
	"log/slog"
	"testing"
	"time"
)

func TestAttrs(t *testing.T) {
	tests := []struct {
		name     string
		attr     slog.Attr
		wantKey  string
		wantKind slog.Kind
		wantVal  any
	}{
		{
			name:     "error message",
			attr:     Err(errors.New("boom")),
			wantKey:  "error",
			wantKind: slog.KindString,
			wantVal:  "boom",
		},
		{
			name:     "address",
			attr:     Addr("[::]:8080"),
			wantKey:  "addr",
			wantKind: slog.KindString,
			wantVal:  "[::]:8080",
		},
		{
			name:     "timeout in go notation",
			attr:     Timeout(10 * time.Second),
			wantKey:  "timeout",
			wantKind: slog.KindString,
			wantVal:  "10s",
		},
		{
			name:     "http method",
			attr:     Method("POST"),
			wantKey:  "method",
			wantKind: slog.KindString,
			wantVal:  "POST",
		},
		{
			name:     "url path",
			attr:     Path("/api/v1/add"),
			wantKey:  "path",
			wantKind: slog.KindString,
			wantVal:  "/api/v1/add",
		},
		{
			name:     "status code is an int",
			attr:     Status(422),
			wantKey:  "status",
			wantKind: slog.KindInt64,
			wantVal:  int64(422),
		},
		{
			name:     "duration in fractional milliseconds",
			attr:     Duration(1500 * time.Microsecond),
			wantKey:  "duration_ms",
			wantKind: slog.KindFloat64,
			wantVal:  1.5,
		},
		{
			name:     "directory",
			attr:     Dir("logs"),
			wantKey:  "dir",
			wantKind: slog.KindString,
			wantVal:  "logs",
		},
		{
			name:     "service name",
			attr:     service(),
			wantKey:  "service",
			wantKind: slog.KindString,
			wantVal:  "calculator-backend",
		},
		{
			name:     "nil error is an empty attr that handlers drop",
			attr:     Err(nil),
			wantKey:  "",
			wantKind: slog.KindAny,
			wantVal:  nil,
		},
		{
			name:     "logger category",
			attr:     category(CategoryHTTP),
			wantKey:  "logger",
			wantKind: slog.KindString,
			wantVal:  "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.Key != tt.wantKey {
				t.Errorf("key = %q, want %q", tt.attr.Key, tt.wantKey)
			}
			if kind := tt.attr.Value.Kind(); kind != tt.wantKind {
				t.Fatalf("kind = %v, want %v", kind, tt.wantKind)
			}
			if got := tt.attr.Value.Any(); got != tt.wantVal {
				t.Errorf("value = %#v, want %#v", got, tt.wantVal)
			}
		})
	}
}
