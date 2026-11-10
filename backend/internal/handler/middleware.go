package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"fullstack-calculator/backend/internal/logger"
)

// AccessLog logs one structured record per request with method, path, status
// and duration. Successful health checks are logged at debug level so
// frequent probes don't flood the log store; failed ones stay visible.
// It also runs for gin's 404/405 fallbacks.
func AccessLog(accessLog *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		path, status := c.Request.URL.Path, c.Writer.Status()
		accessLog.LogAttrs(c.Request.Context(), accessLogLevel(path, status), msgRequestCompleted,
			logger.Method(c.Request.Method),
			logger.Path(path),
			logger.Status(status),
			logger.Duration(time.Since(start)),
		)
	}
}

func accessLogLevel(path string, status int) slog.Level {
	if path == PathHealth && status < http.StatusBadRequest {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
