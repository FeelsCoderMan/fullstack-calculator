// Package handler is the HTTP layer: routing, request decoding, and response
// writing. It is the only package that depends on gin.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter registers all HTTP routes on a gin engine behind RequestLogger,
// serving the operations with calc.
//
// It returns an http.Handler rather than *gin.Engine so that callers (the
// server wiring) stay framework-agnostic. The engine is built with gin.New,
// not gin.Default: gin's own logger and recovery middleware write plain text,
// while RequestLogger writes one JSON record per request and recovers panics
// itself (docs/logging.md).
func NewRouter(accessLog *slog.Logger, calc Calculator) http.Handler {
	// Debug mode prints plain-text route listings and warnings to stdout,
	// which would break the JSON-lines log stream.
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.HandleMethodNotAllowed = true // 405 instead of 404 for a known path with the wrong method
	r.Use(RequestLogger(accessLog))
	r.NoRoute(notFound)
	r.NoMethod(methodNotAllowed)

	r.Match([]string{http.MethodGet, http.MethodHead}, PathHealth, Health)
	r.POST(PathAdd, binary(calc.Add))
	r.POST(PathSubtract, binary(calc.Subtract))
	r.POST(PathMultiply, binary(calc.Multiply))
	r.POST(PathDivide, binary(calc.Divide))
	r.POST(PathPower, binary(calc.Power))
	r.POST(PathPercentage, binary(calc.Percentage))
	r.POST(PathSqrt, unary(calc.Sqrt))

	// API documentation: Swagger UI and the raw spec (api/openapi.yaml).
	r.GET(PathSwaggerUI, SwaggerUI)
	r.GET(PathSwaggerSpec, OpenAPISpec)
	return r
}
