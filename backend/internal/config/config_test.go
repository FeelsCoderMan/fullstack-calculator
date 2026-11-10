package config

import (
	"errors"
	"log/slog"
	"testing"
	"time"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadServer(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Server
		wantErr error
	}{
		{
			name: "all variables set",
			env:  map[string]string{"PORT": "9090", "SHUTDOWN_TIMEOUT": "3s"},
			want: Server{Port: 9090, ShutdownTimeout: 3 * time.Second},
		},
		{
			name: "port zero is allowed for tests",
			env:  map[string]string{"PORT": "0", "SHUTDOWN_TIMEOUT": "10s"},
			want: Server{Port: 0, ShutdownTimeout: 10 * time.Second},
		},
		{
			name:    "no variables set",
			env:     map[string]string{},
			wantErr: ErrMissingEnv,
		},
		{
			name:    "missing port",
			env:     map[string]string{"SHUTDOWN_TIMEOUT": "10s"},
			wantErr: ErrMissingEnv,
		},
		{
			name:    "empty port counts as missing",
			env:     map[string]string{"PORT": "", "SHUTDOWN_TIMEOUT": "10s"},
			wantErr: ErrMissingEnv,
		},
		{
			name:    "missing shutdown timeout",
			env:     map[string]string{"PORT": "8080"},
			wantErr: ErrMissingEnv,
		},
		{
			name:    "non-numeric port",
			env:     map[string]string{"PORT": "abc", "SHUTDOWN_TIMEOUT": "10s"},
			wantErr: ErrInvalidEnv,
		},
		{
			name:    "negative port",
			env:     map[string]string{"PORT": "-1", "SHUTDOWN_TIMEOUT": "10s"},
			wantErr: ErrInvalidEnv,
		},
		{
			name:    "port above 65535",
			env:     map[string]string{"PORT": "65536", "SHUTDOWN_TIMEOUT": "10s"},
			wantErr: ErrInvalidEnv,
		},
		{
			name:    "timeout that is not a duration",
			env:     map[string]string{"PORT": "8080", "SHUTDOWN_TIMEOUT": "soon"},
			wantErr: ErrInvalidEnv,
		},
		{
			name:    "zero timeout",
			env:     map[string]string{"PORT": "8080", "SHUTDOWN_TIMEOUT": "0s"},
			wantErr: ErrInvalidEnv,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadServer(envFrom(tt.env))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("LoadServer(%v) error = %v, want %v", tt.env, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("LoadServer(%v) = %+v, want %+v", tt.env, got, tt.want)
			}
		})
	}
}

func TestLoadLog(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Log
		wantErr error
	}{
		{
			name: "all variables set",
			env:  map[string]string{"LOG_LEVEL": "DEBUG", "LOG_TO_FILE": "true", "LOG_DIR": "/tmp/calc-logs"},
			want: Log{Level: slog.LevelDebug, ToFile: true, Dir: "/tmp/calc-logs"},
		},
		{
			name: "log level with slog offset",
			env:  map[string]string{"LOG_LEVEL": "info+2", "LOG_TO_FILE": "false", "LOG_DIR": "logs"},
			want: Log{Level: slog.LevelInfo + 2, ToFile: false, Dir: "logs"},
		},
		{
			name:    "no variables set",
			env:     map[string]string{},
			wantErr: ErrMissingEnv,
		},
		{
			name:    "missing log level",
			env:     map[string]string{"LOG_TO_FILE": "false", "LOG_DIR": "logs"},
			wantErr: ErrMissingEnv,
		},
		{
			name:    "missing log to file",
			env:     map[string]string{"LOG_LEVEL": "info", "LOG_DIR": "logs"},
			wantErr: ErrMissingEnv,
		},
		{
			name:    "missing log dir even when file logging is off",
			env:     map[string]string{"LOG_LEVEL": "info", "LOG_TO_FILE": "false"},
			wantErr: ErrMissingEnv,
		},
		{
			name:    "unknown log level",
			env:     map[string]string{"LOG_LEVEL": "verbose", "LOG_TO_FILE": "false", "LOG_DIR": "logs"},
			wantErr: ErrInvalidEnv,
		},
		{
			name:    "log to file that is not a boolean",
			env:     map[string]string{"LOG_LEVEL": "info", "LOG_TO_FILE": "yes please", "LOG_DIR": "logs"},
			wantErr: ErrInvalidEnv,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadLog(envFrom(tt.env))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("LoadLog(%v) error = %v, want %v", tt.env, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("LoadLog(%v) = %+v, want %+v", tt.env, got, tt.want)
			}
		})
	}
}

func TestServerAddr(t *testing.T) {
	tests := []struct {
		name string
		port int
		want string
	}{
		{
			name: "typical port",
			port: 8080,
			want: ":8080",
		},
		{
			name: "os-assigned port",
			port: 0,
			want: ":0",
		},
		{
			name: "highest port",
			port: 65535,
			want: ":65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Server{Port: tt.port}).Addr(); got != tt.want {
				t.Errorf("Server{Port: %d}.Addr() = %q, want %q", tt.port, got, tt.want)
			}
		})
	}
}
