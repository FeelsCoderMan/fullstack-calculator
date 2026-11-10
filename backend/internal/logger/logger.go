// Package logger builds structured slog loggers that emit one JSON object per
// line on stdout (for Loki), optionally mirrored to per-category files in
// development.
package logger

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Options configures a Factory.
type Options struct {
	Level slog.Leveler
	// ToFile mirrors every record to Dir/<category>-<timestamp>.log. Development only.
	ToFile bool
	Dir    string
	// Stdout receives every record; typically os.Stdout.
	Stdout io.Writer
	// Now returns the current time; used for the file name timestamp. Defaults to time.Now.
	Now func() time.Time
}

// Factory hands out one logger per category. It owns any open log files and
// must be closed on shutdown.
type Factory struct {
	loggers map[Category]*slog.Logger
	stdout  io.Writer
	level   slog.Leveler
	files   []*os.File
}

// NewFactory creates loggers for every category. When opts.ToFile is set it
// creates opts.Dir and opens one file per category up front, so a bad
// directory fails at startup rather than on first log.
func NewFactory(opts Options) (*Factory, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	f := &Factory{
		loggers: make(map[Category]*slog.Logger),
		stdout:  &syncWriter{w: opts.Stdout},
		level:   opts.Level,
	}

	if opts.ToFile {
		if err := os.MkdirAll(opts.Dir, dirPerm); err != nil {
			return nil, fmt.Errorf("create log dir %s: %w", opts.Dir, err)
		}
	}

	startedAt := now()
	for _, c := range Categories() {
		w := f.stdout
		if opts.ToFile {
			path := filepath.Join(opts.Dir, FileName(c, startedAt))
			file, err := os.OpenFile(path, fileFlags, filePerm)
			if err != nil {
				_ = f.Close()
				return nil, fmt.Errorf("open log file %s: %w", path, err)
			}
			f.files = append(f.files, file)
			w = io.MultiWriter(f.stdout, file)
		}
		f.loggers[c] = newLogger(w, opts.Level, c)
	}
	return f, nil
}

// Logger returns the logger for a category. Unknown categories log to stdout only.
func (f *Factory) Logger(c Category) *slog.Logger {
	if l, ok := f.loggers[c]; ok {
		return l
	}
	return newLogger(f.stdout, f.level, c)
}

// Close closes any open log files.
func (f *Factory) Close() error {
	var errs []error
	for _, file := range f.files {
		errs = append(errs, file.Close())
	}
	f.files = nil
	return errors.Join(errs...)
}

// FileName returns the log file name for a category, stamped with t in UTC,
// e.g. "http-20261110T191503Z.log".
func FileName(c Category, t time.Time) string {
	return fmt.Sprintf(fileNameFormat, c, t.UTC().Format(fileTimeLayout))
}

// NewStdout returns a stdout-only logger for a category. Use it only where no
// Factory exists yet (e.g. reporting a config error at startup).
func NewStdout(w io.Writer, c Category) *slog.Logger {
	return newLogger(&syncWriter{w: w}, slog.LevelInfo, c)
}

func newLogger(w io.Writer, level slog.Leveler, c Category) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(h.WithAttrs([]slog.Attr{service(), category(c)}))
}

// syncWriter serializes writes so records from different category loggers
// sharing one stdout never interleave mid-line.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
