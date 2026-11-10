package handler

// URL paths served outside the versioned API.
const (
	PathHealth      = "/health"
	PathSwaggerUI   = "/swagger/"
	PathSwaggerSpec = "/swagger/openapi.yaml"
)

// Route groups of the API: every operation is served under PathAPI + PathV1.
const (
	PathAPI = "/api"
	PathV1  = "/v1"
)

// Operation routes, relative to the /api/v1 group (routes.go registers them
// there).
const (
	routeAdd        = "/add"
	routeSubtract   = "/subtract"
	routeMultiply   = "/multiply"
	routeDivide     = "/divide"
	routePower      = "/power"
	routePercentage = "/percentage"
	routeSqrt       = "/sqrt"
)

// Full URL paths of the operations, as clients call them (api/openapi.yaml).
const (
	PathAdd        = PathAPI + PathV1 + routeAdd
	PathSubtract   = PathAPI + PathV1 + routeSubtract
	PathMultiply   = PathAPI + PathV1 + routeMultiply
	PathDivide     = PathAPI + PathV1 + routeDivide
	PathPower      = PathAPI + PathV1 + routePower
	PathPercentage = PathAPI + PathV1 + routePercentage
	PathSqrt       = PathAPI + PathV1 + routeSqrt
)

// Request body limits and properties.
const (
	// maxBodyBytes is the largest accepted request body (1 MiB).
	maxBodyBytes = 1 << 20

	fieldLeftOperand  = "leftOperand"
	fieldRightOperand = "rightOperand"
	fieldOperand      = "operand"
	fieldPrecision    = "precision"
)

// bitSize makes strconv parse numbers as float64.
const bitSize = 64

// Content types of the documentation endpoints.
const (
	contentTypeHTML = "text/html; charset=utf-8"
	contentTypeYAML = "application/yaml"
)

// Request correlation.
const (
	// HeaderRequestID carries the request ID in both directions.
	HeaderRequestID = "X-Request-ID"
	// requestIDPattern is what an incoming X-Request-ID must look like to be
	// reused; anything else (empty, too long, odd characters that could be
	// used for log injection) is replaced by a generated ID.
	requestIDPattern = `^[A-Za-z0-9-]{1,64}$`
)

// Gin context keys.
const (
	// contextKeyFailure holds the failure recorded by writeError, read once by
	// RequestLogger when it writes the request record.
	contextKeyFailure = "handler.failure"
)

// Log messages.
const (
	msgRequestCompleted = "request completed"
)
