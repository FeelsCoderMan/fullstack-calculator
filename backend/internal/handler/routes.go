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

	// Calculator operations: /api/v1/<operation>. A breaking change gets a new
	// version group (/api/v2) next to this one instead of changing it.
	api := r.Group(PathAPI)
	v1 := api.Group(PathV1)
	v1.POST(routeAdd, binary(calc.Add))
	v1.POST(routeSubtract, binary(calc.Subtract))
	v1.POST(routeMultiply, binary(calc.Multiply))
	v1.POST(routeDivide, binary(calc.Divide))
	v1.POST(routePower, binary(calc.Power))
	v1.POST(routePercentage, binary(calc.Percentage))
	v1.POST(routeSqrt, unary(calc.Sqrt))

	// API documentation: Swagger UI and the raw spec (api/openapi.yaml).
	r.GET(PathSwaggerUI, SwaggerUI)
	r.GET(PathSwaggerSpec, OpenAPISpec)
	return r
}
