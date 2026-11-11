// Package handler is the HTTP layer: routing, request binding and validation,
// and response writing. Together with middleware, it is the only package that
// depends on gin.
package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"fullstack-calculator/backend/internal/middleware"
)

// NewRouter registers all HTTP routes on a gin engine behind
// middleware.RequestLogger, serving the operations with calc.
//
// It returns an http.Handler rather than *gin.Engine so that callers (the
// server wiring) stay framework-agnostic.
func NewRouter(accessLog *slog.Logger, calc Calculator) (http.Handler, error) {
	validate, err := newValidator()
	if err != nil {
		return nil, fmt.Errorf("new validator: %w", err)
	}

	r := newEngine(accessLog)
	r.Match([]string{http.MethodGet, http.MethodHead}, pathHealth, health)

	// Calculator operations: /api/v1/<operation>. A breaking change gets a new
	// version group (/api/v2) next to this one instead of changing it.
	v1 := r.Group(pathAPI).Group(pathV1)
	newCalculatorHandler(calc, validate).registerHandlers(v1)

	// API documentation: Swagger UI and the raw spec (api/openapi.yaml).
	r.GET(pathSwaggerUI, swaggerUI)
	r.GET(pathSwaggerSpec, openAPISpec)
	return r, nil
}

// newEngine returns a gin engine with no routes yet, set up the way every
// request is handled: middleware.RequestLogger in front (answering panics with
// writeError, logging successful health probes at debug), JSON 404/405
// fallbacks, and 405 instead of 404 for a known path with the wrong method.
//
// It is built with gin.New, not gin.Default: gin's own logger and recovery
// middleware write plain text, while middleware.RequestLogger writes one JSON
// record per request and recovers panics itself (docs/logging.md). Tests build
// their engines here too, so they run behind exactly what NewRouter uses.
func newEngine(accessLog *slog.Logger) *gin.Engine {
	// Debug mode prints plain-text route listings and warnings to stdout,
	// which would break the JSON-lines log stream.
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(middleware.RequestLogger(accessLog, writeError, pathHealth))
	r.NoRoute(notFound)
	r.NoMethod(methodNotAllowed)
	return r
}
