package logger

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var ctx = context.Background()

var fixedTime = time.Date(2026, 11, 10, 16, 15, 3, 0, time.UTC)

func fixedNow() time.Time { return fixedTime }

func decodeLines(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	var records []map[string]any
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var rec map[string]any
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("line is not valid JSON: %q: %v", sc.Text(), err)
		}
		records = append(records, rec)
	}
	return records
}

// assertRecords checks that got has one record per want, in order, and that
// every key in each want matches (other keys, such as time, are ignored).
func assertRecords(t *testing.T, source string, got, want []map[string]any) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d records, want %d: %v", source, len(got), len(want), got)
	}
	for i := range want {
		for key, wantVal := range want[i] {
			if got[i][key] != wantVal {
				t.Errorf("%s: record[%d][%q] = %#v, want %#v", source, i, key, got[i][key], wantVal)
			}
		}
	}
}

func TestFileName(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		time     time.Time
		want     string
	}{
		{
			name:     "utc time is used as is",
			category: CategoryHTTP,
			time:     time.Date(2026, 11, 10, 16, 15, 3, 0, time.UTC),
			want:     "http-20261110T161503Z.log",
		},
		{
			name:     "local time is converted to utc",
			category: CategoryServer,
			time:     time.Date(2026, 11, 10, 19, 15, 3, 0, time.FixedZone("UTC+3", 3*60*60)),
			want:     "server-20261110T161503Z.log",
		},
		{
			name:     "conversion to utc can move the date back",
			category: CategoryHTTP,
			time:     time.Date(2026, 11, 11, 1, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60)),
			want:     "http-20261110T220000Z.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileName(tt.category, tt.time); got != tt.want {
				t.Errorf("FileName(%q, %v) = %q, want %q", tt.category, tt.time, got, tt.want)
			}
		})
	}
}

func TestFactoryOutput(t *testing.T) {
	tests := []struct {
		name        string
		toFile      bool
		level       slog.Level
		category    Category
		recordLevel slog.Level
		msg         string
		attrs       []slog.Attr
		wantStdout  []map[string]any
		wantFiles   map[string]bool // file name → contains the record; nil = no log dir created
	}{
		{
			name:        "stdout only",
			toFile:      false,
			level:       slog.LevelInfo,
			category:    CategoryServer,
			recordLevel: slog.LevelInfo,
			msg:         "server listening",
			attrs:       []slog.Attr{Addr(":8080")},
			wantStdout: []map[string]any{
				{"level": "INFO", "msg": "server listening", "service": "calculator-backend", "logger": "server", "addr": ":8080"},
			},
			wantFiles: nil,
		},
		{
			name:        "record below the level is dropped",
			toFile:      false,
			level:       slog.LevelInfo,
			category:    CategoryServer,
			recordLevel: slog.LevelDebug,
			msg:         "filtered out",
			wantStdout:  nil,
			wantFiles:   nil,
		},
		{
			name:        "file output mirrors stdout into the category's file",
			toFile:      true,
			level:       slog.LevelDebug,
			category:    CategoryHTTP,
			recordLevel: slog.LevelInfo,
			msg:         "request completed",
			attrs:       []slog.Attr{Status(200)},
			wantStdout: []map[string]any{
				{"level": "INFO", "msg": "request completed", "service": "calculator-backend", "logger": "http", "status": float64(200)},
			},
			wantFiles: map[string]bool{
				"http-20261110T161503Z.log":   true,
				"server-20261110T161503Z.log": false,
			},
		},
		{
			name:        "unknown category goes to stdout only",
			toFile:      true,
			level:       slog.LevelInfo,
			category:    Category("custom"),
			recordLevel: slog.LevelInfo,
			msg:         "hello",
			wantStdout: []map[string]any{
				{"level": "INFO", "msg": "hello", "service": "calculator-backend", "logger": "custom"},
			},
			wantFiles: map[string]bool{
				"http-20261110T161503Z.log":   false,
				"server-20261110T161503Z.log": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			dir := filepath.Join(t.TempDir(), "logs")

			f, err := NewFactory(Options{Level: tt.level, ToFile: tt.toFile, Dir: dir, Stdout: &stdout, Now: fixedNow})
			if err != nil {
				t.Fatalf("NewFactory() error = %v", err)
			}
			f.Logger(tt.category).LogAttrs(ctx, tt.recordLevel, tt.msg, tt.attrs...)
			if err := f.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}

			assertRecords(t, "stdout", decodeLines(t, stdout.Bytes()), tt.wantStdout)

			entries, err := os.ReadDir(dir)
			if tt.wantFiles == nil {
				if !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("log dir exists (ReadDir error = %v), want it not created", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("read log dir: %v", err)
			}
			if len(entries) != len(tt.wantFiles) {
				t.Errorf("got %d log files, want %d", len(entries), len(tt.wantFiles))
			}
			for name, hasRecord := range tt.wantFiles {
				data, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatalf("read %s: %v", name, err)
				}
				var want []map[string]any
				if hasRecord {
					want = tt.wantStdout
				}
				assertRecords(t, name, decodeLines(t, data), want)
			}
		})
	}
}

func TestNewFactoryErrors(t *testing.T) {
	tests := []struct {
		name    string
		dir     func(t *testing.T) string
		wantErr error
	}{
		{
			name: "log dir under a regular file",
			dir: func(t *testing.T) string {
				t.Helper()
				blocker := filepath.Join(t.TempDir(), "file")
				if err := os.WriteFile(blocker, nil, 0o644); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(blocker, "logs")
			},
			wantErr: syscall.ENOTDIR,
		},
		{
			name: "a later category file cannot be opened",
			dir: func(t *testing.T) string {
				t.Helper()
				// A directory where the http log file should go makes its OpenFile
				// fail after the server file has already been opened.
				dir := t.TempDir()
				if err := os.Mkdir(filepath.Join(dir, "http-20261110T161503Z.log"), 0o755); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: syscall.EISDIR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFactory(Options{ToFile: true, Dir: tt.dir(t), Stdout: &bytes.Buffer{}, Now: fixedNow})
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewFactory() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewStdout(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		want     map[string]any
	}{
		{
			name:     "server category",
			category: CategoryServer,
			want:     map[string]any{"level": "ERROR", "msg": "startup failed", "service": "calculator-backend", "logger": "server"},
		},
		{
			name:     "http category",
			category: CategoryHTTP,
			want:     map[string]any{"level": "ERROR", "msg": "startup failed", "service": "calculator-backend", "logger": "http"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			NewStdout(&buf, tt.category).LogAttrs(ctx, slog.LevelError, "startup failed")
			assertRecords(t, "stdout", decodeLines(t, buf.Bytes()), []map[string]any{tt.want})
		})
	}
}

// Concurrency test: straight-line by design (see the table-driven-tests skill, rule 11).
func TestConcurrentCategoriesDoNotInterleave(t *testing.T) {
	var buf bytes.Buffer
	f, err := NewFactory(Options{Level: slog.LevelInfo, Stdout: &buf})
	if err != nil {
		t.Fatal(err)
	}
	payload := strings.Repeat("x", 8192) // larger than a pipe's atomic write size

	var wg sync.WaitGroup
	for _, c := range Categories() {
		wg.Add(1)
		go func(c Category) {
			defer wg.Done()
			for range 50 {
				f.Logger(c).LogAttrs(ctx, slog.LevelInfo, "bulk", Path(payload))
			}
		}(c)
	}
	wg.Wait()

	if recs := decodeLines(t, buf.Bytes()); len(recs) != 50*len(Categories()) {
		t.Errorf("got %d valid records, want %d", len(recs), 50*len(Categories()))
	}
}
