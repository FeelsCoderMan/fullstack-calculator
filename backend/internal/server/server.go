// Package server wires configuration, logging and the HTTP handler into a
// runnable App with graceful shutdown.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fullstack-calculator/backend/internal/config"
	"fullstack-calculator/backend/internal/handler"
	"fullstack-calculator/backend/internal/logger"
	"fullstack-calculator/backend/internal/service"
)

// handlerFactory builds the root HTTP handler from the configured loggers.
type handlerFactory func(logs *logger.Factory) (http.Handler, error)

func newRouter(logs *logger.Factory) (http.Handler, error) {
	return handler.NewRouter(logs.Logger(logger.CategoryHTTP), service.NewCalculator())
}

// App is a configured HTTP server that is already listening. Call Run exactly
// once to serve requests until shutdown; Run releases every resource.
type App struct {
	shutdownTimeout time.Duration
	logs            *logger.Factory
	serverLog       *slog.Logger
	srv             *http.Server
	ln              net.Listener
}

// NewApp loads configuration from the environment, sets up logging to stdout
// (and to per-category files when LOG_TO_FILE=true) and binds the listen
// address, so every startup problem surfaces here rather than in Run.
func NewApp() (*App, error) {
	return newApp(os.Getenv, os.Stdout, newRouter)
}

func newApp(getenv func(string) string, stdout io.Writer, newHandler handlerFactory) (*App, error) {
	// Both configs are loaded before anything is opened, so a bad env var
	// fails fast with nothing to clean up.
	serverCfg, err := config.LoadServer(getenv)
	if err != nil {
		return nil, fmt.Errorf("load server config: %w", err)
	}
	logCfg, err := config.LoadLog(getenv)
	if err != nil {
		return nil, fmt.Errorf("load log config: %w", err)
	}

	logs, err := logger.NewFactory(logger.Options{
		Level:  logCfg.Level,
		ToFile: logCfg.ToFile,
		Dir:    logCfg.Dir,
		Stdout: stdout,
	})
	if err != nil {
		return nil, fmt.Errorf("init logging: %w", err)
	}

	serverLog := logs.Logger(logger.CategoryServer)
	if logCfg.ToFile {
		serverLog.LogAttrs(context.Background(), slog.LevelWarn, msgFileLoggingEnabled, logger.Dir(logCfg.Dir))
	}

	h, err := newHandler(logs)
	if err != nil {
		return nil, closeLogs(logs, fmt.Errorf("build handler: %w", err))
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), network, serverCfg.Addr())
	if err != nil {
		return nil, closeLogs(logs, fmt.Errorf("listen on %s: %w", serverCfg.Addr(), err))
	}

	return &App{
		shutdownTimeout: serverCfg.ShutdownTimeout,
		logs:            logs,
		serverLog:       serverLog,
		ln:              ln,
		srv: &http.Server{
			Handler:           h,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			ErrorLog:          slog.NewLogLogger(logs.Logger(logger.CategoryHTTP).Handler(), slog.LevelError),
		},
	}, nil
}

// closeLogs closes the log files after a failed startup and returns err,
// joined with the close error if there is one.
func closeLogs(logs *logger.Factory, err error) error {
	if closeErr := logs.Close(); closeErr != nil {
		return errors.Join(err, fmt.Errorf("close log files: %w", closeErr))
	}
	return err
}

// Addr returns the bound listen address (useful with PORT=0).
func (a *App) Addr() string { return a.ln.Addr().String() }

// Run serves requests until ctx is cancelled or SIGINT/SIGTERM arrives, then
// shuts down gracefully, letting in-flight requests finish within the
// configured timeout; anything still running after that is force-closed.
// A second signal during the drain kills the process immediately.
func (a *App) Run(ctx context.Context) (err error) {
	// Records logged after Close (by handlers still running after a forced
	// shutdown) still reach stdout; only the dev log file misses them.
	defer func() {
		if closeErr := a.logs.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close log files: %w", closeErr))
		}
	}()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	// After the first signal, restore default handling so a second Ctrl+C
	// kills the process immediately instead of waiting for the drain.
	go func() {
		<-ctx.Done()
		stop()
	}()

	a.serverLog.LogAttrs(ctx, slog.LevelInfo, msgServerListening, logger.Addr(a.Addr()))

	serveErr := make(chan error, 1)
	go func() {
		if err := a.srv.Serve(a.ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	a.serverLog.LogAttrs(ctx, slog.LevelInfo, msgShutdownStarted, logger.Timeout(a.shutdownTimeout))
	// ctx is already cancelled here; keep its values but not its cancellation.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.shutdownTimeout)
	defer cancel()

	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		_ = a.srv.Close() // force-close what didn't drain; the caller logs the returned error once
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	a.serverLog.LogAttrs(ctx, slog.LevelInfo, msgServerStopped)
	return nil
}
