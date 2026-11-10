package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"fullstack-calculator/backend/internal/config"
	"fullstack-calculator/backend/internal/logger"
)

const testWait = 5 * time.Second

var testClient = &http.Client{Timeout: testWait}

// get sends a GET request bound to the test's context.
func get(t *testing.T, url string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	return testClient.Do(req)
}

// testEnv returns a getenv with every required variable set to a valid
// value (random port, stdout-only logging), with overrides applied on top.
// An override of "" makes that variable missing.
func testEnv(overrides map[string]string) func(string) string {
	env := map[string]string{
		"PORT":             "0",
		"SHUTDOWN_TIMEOUT": "2s",
		"LOG_LEVEL":        "info",
		"LOG_TO_FILE":      "false",
		"LOG_DIR":          "logs",
	}
	maps.Copy(env, overrides)
	return func(k string) string { return env[k] }
}

// fixedHandler ignores the logger factory and serves h.
func fixedHandler(h http.Handler) handlerFactory {
	return func(*logger.Factory) http.Handler { return h }
}

// startServer builds an App from testEnv(env) (a random port unless env overrides PORT), runs it
// in a goroutine and returns its address, a cancel func that triggers
// shutdown, and a channel with Run's result. The listener is bound before
// newApp returns, so requests can be sent right away.
func startServer(t *testing.T, newHandler handlerFactory, env map[string]string, stdout io.Writer) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	app, err := newApp(testEnv(env), stdout, newHandler)
	if err != nil {
		t.Fatalf("newApp() error = %v, want nil", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()
	return app.Addr(), cancel, done
}

func waitDone(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(testWait):
		t.Fatal("Run() did not return")
		return nil
	}
}

func TestRunEndToEnd(t *testing.T) {
	tests := []struct {
		name         string
		env          map[string]string
		path         string
		wantStatus   int
		wantStdout   []string
		wantLogFiles int
	}{
		{
			name:       "health check with stdout logging only",
			env:        map[string]string{},
			path:       "/health",
			wantStatus: http.StatusNoContent,
			wantStdout: []string{
				`"msg":"server listening"`,
				`"msg":"server stopped"`,
			},
			wantLogFiles: 0,
		},
		{
			name:       "unknown route with file logging enabled",
			env:        map[string]string{"LOG_TO_FILE": "true"},
			path:       "/missing",
			wantStatus: http.StatusNotFound,
			wantStdout: []string{
				`"msg":"file logging enabled; development only"`,
				`"msg":"server listening"`,
				`"logger":"http","method":"GET","path":"/missing","status":404`,
				`"msg":"server stopped"`,
			},
			wantLogFiles: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "logs")
			env := map[string]string{"LOG_DIR": dir, "SHUTDOWN_TIMEOUT": "2s"}
			maps.Copy(env, tt.env)
			var stdout bytes.Buffer

			addr, cancel, done := startServer(t, newRouter, env, &stdout)
			resp, err := get(t, "http://"+addr+tt.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tt.path, err)
			}
			resp.Body.Close()
			cancel()
			if err := waitDone(t, done); err != nil {
				t.Fatalf("Run() error = %v, want nil", err)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("GET %s status = %d, want %d", tt.path, resp.StatusCode, tt.wantStatus)
			}
			for _, want := range tt.wantStdout {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("stdout missing %s:\n%s", want, stdout.String())
				}
			}

			entries, err := os.ReadDir(dir)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("read log dir: %v", err)
			}
			if len(entries) != tt.wantLogFiles {
				t.Errorf("got %d log files, want %d", len(entries), tt.wantLogFiles)
			}
			for _, e := range entries {
				if info, err := e.Info(); err != nil || info.Size() == 0 {
					t.Errorf("log file %s is empty or unreadable (err = %v)", e.Name(), err)
				}
			}
		})
	}
}

// blockingHandler signals when a request arrives and waits for release before responding.
func blockingHandler(started chan<- struct{}, release <-chan struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		started <- struct{}{}
		<-release
		w.WriteHeader(http.StatusOK)
	})
}

func TestRunShutdown(t *testing.T) {
	tests := []struct {
		name            string
		shutdownTimeout string
		releaseRequest  bool // let the in-flight request finish after shutdown begins
		wantErr         error
		wantRequestOK   bool
	}{
		{
			name:            "in-flight request drains before exit",
			shutdownTimeout: "5s",
			releaseRequest:  true,
			wantErr:         nil,
			wantRequestOK:   true,
		},
		{
			name:            "request outliving the timeout is force-closed",
			shutdownTimeout: "100ms",
			releaseRequest:  false,
			wantErr:         context.DeadlineExceeded,
			wantRequestOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			started, release := make(chan struct{}, 1), make(chan struct{})
			var releaseOnce sync.Once
			releaseFn := func() { releaseOnce.Do(func() { close(release) }) }
			t.Cleanup(releaseFn)

			addr, cancel, done := startServer(t, fixedHandler(blockingHandler(started, release)),
				map[string]string{"SHUTDOWN_TIMEOUT": tt.shutdownTimeout}, io.Discard)

			requestErr := make(chan error, 1)
			go func() {
				resp, err := get(t, "http://"+addr+"/")
				if err != nil {
					requestErr <- err
					return
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					requestErr <- fmt.Errorf("status %d", resp.StatusCode)
					return
				}
				requestErr <- nil
			}()

			select {
			case <-started:
			case <-time.After(testWait):
				t.Fatal("request never reached the handler")
			}
			cancel() // shutdown begins while the request is in flight
			if tt.releaseRequest {
				time.Sleep(200 * time.Millisecond) // let Shutdown start before the handler returns
				releaseFn()
			}

			if err := waitDone(t, done); !errors.Is(err, tt.wantErr) {
				t.Errorf("Run() error = %v, want %v", err, tt.wantErr)
			}
			select {
			case err := <-requestErr:
				if gotOK := err == nil; gotOK != tt.wantRequestOK {
					t.Errorf("in-flight request ok = %v (err = %v), want %v", gotOK, err, tt.wantRequestOK)
				}
			case <-time.After(testWait):
				t.Fatal("in-flight request never completed")
			}
		})
	}
}

func TestNewAppStartupErrors(t *testing.T) {
	tests := []struct {
		name    string
		env     func(t *testing.T) map[string]string
		wantErr error
	}{
		{
			name: "invalid port",
			env: func(*testing.T) map[string]string {
				return map[string]string{"PORT": "nope"}
			},
			wantErr: config.ErrInvalidEnv,
		},
		{
			name: "missing server variable",
			env: func(*testing.T) map[string]string {
				return map[string]string{"SHUTDOWN_TIMEOUT": ""}
			},
			wantErr: config.ErrMissingEnv,
		},
		{
			name: "missing log variable",
			env: func(*testing.T) map[string]string {
				return map[string]string{"LOG_DIR": ""}
			},
			wantErr: config.ErrMissingEnv,
		},
		{
			name: "invalid log level",
			env: func(*testing.T) map[string]string {
				return map[string]string{"LOG_LEVEL": "verbose"}
			},
			wantErr: config.ErrInvalidEnv,
		},
		{
			name: "port already in use",
			env: func(t *testing.T) map[string]string {
				t.Helper()
				var lc net.ListenConfig
				ln, err := lc.Listen(t.Context(), "tcp", ":0")
				if err != nil {
					t.Fatalf("reserve port: %v", err)
				}
				t.Cleanup(func() { _ = ln.Close() })
				tcpAddr, ok := ln.Addr().(*net.TCPAddr)
				if !ok {
					t.Fatalf("listener addr = %T, want *net.TCPAddr", ln.Addr())
				}
				return map[string]string{"PORT": strconv.Itoa(tcpAddr.Port)}
			},
			wantErr: syscall.EADDRINUSE,
		},
		{
			name: "log dir under a regular file",
			env: func(t *testing.T) map[string]string {
				t.Helper()
				blocker := filepath.Join(t.TempDir(), "file")
				if err := os.WriteFile(blocker, nil, 0o644); err != nil {
					t.Fatal(err)
				}
				return map[string]string{"LOG_TO_FILE": "true", "LOG_DIR": filepath.Join(blocker, "logs")}
			},
			wantErr: syscall.ENOTDIR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := newApp(testEnv(tt.env(t)), io.Discard, newRouter)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("newApp() error = %v, want %v", err, tt.wantErr)
			}
			if app != nil {
				t.Errorf("newApp() app = %v, want nil on error", app)
			}
		})
	}
}
