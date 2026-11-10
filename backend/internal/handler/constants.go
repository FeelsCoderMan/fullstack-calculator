package handler

// URL paths served outside the versioned API.
const (
	PathHealth      = "/health"
	PathSwaggerUI   = "/swagger/"
	PathSwaggerSpec = "/swagger/openapi.yaml"
)

// URL paths of the versioned API (api/openapi.yaml).
const (
	PathAdd        = "/api/v1/add"
	PathSubtract   = "/api/v1/subtract"
	PathMultiply   = "/api/v1/multiply"
	PathDivide     = "/api/v1/divide"
	PathPower      = "/api/v1/power"
	PathPercentage = "/api/v1/percentage"
	PathSqrt       = "/api/v1/sqrt"
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
