// Package handler is the HTTP layer: routing, request decoding, and response
// writing. It is the only package that depends on gin.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter registers all HTTP routes on a gin engine with access logging.
//
// It returns an http.Handler rather than *gin.Engine so that callers (the
// server wiring) stay framework-agnostic. The engine is built with gin.New,
// not gin.Default: gin's own logger and recovery middleware write plain text,
// while all logs here are JSON lines from AccessLog. A panicking handler is
// still recovered and logged by net/http via http.Server.ErrorLog.
func NewRouter(accessLog *slog.Logger) http.Handler {
	// Debug mode prints plain-text route listings and warnings to stdout,
	// which would break the JSON-lines log stream.
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.HandleMethodNotAllowed = true // 405 instead of 404 for a known path with the wrong method
	r.Use(AccessLog(accessLog))

	r.Match([]string{http.MethodGet, http.MethodHead}, PathHealth, Health)
	return r
}
