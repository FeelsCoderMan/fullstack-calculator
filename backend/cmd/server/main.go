// Command server runs the calculator HTTP API.
package main

import (
	"context"
	"log/slog"
	"os"

	"fullstack-calculator/backend/internal/logger"
	"fullstack-calculator/backend/internal/server"
)

func main() {
	ctx := context.Background()
	// No Factory exists before NewApp succeeds, so failures go through a
	// stdout-only logger.
	errLog := logger.NewStdout(os.Stdout, logger.CategoryServer)

	app, err := server.NewApp()
	if err != nil {
		errLog.LogAttrs(ctx, slog.LevelError, msgStartupFailed, logger.Err(err))
		os.Exit(1)
	}
	if err := app.Run(ctx); err != nil {
		errLog.LogAttrs(ctx, slog.LevelError, msgServerFailed, logger.Err(err))
		os.Exit(1)
	}
}
