package handler

// URL paths served outside the versioned API.
const (
	pathHealth      = "/health"
	pathSwaggerUI   = "/swagger/"
	pathSwaggerSpec = "/swagger/openapi.yaml"
)

// Route groups of the API: every operation is served under pathAPI + pathV1.
const (
	pathAPI = "/api"
	pathV1  = "/v1"
)

// Operation routes, relative to the /api/v1 group
// (calculatorHandler.registerHandlers registers them there).
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
	pathAdd        = pathAPI + pathV1 + routeAdd
	pathSubtract   = pathAPI + pathV1 + routeSubtract
	pathMultiply   = pathAPI + pathV1 + routeMultiply
	pathDivide     = pathAPI + pathV1 + routeDivide
	pathPower      = pathAPI + pathV1 + routePower
	pathPercentage = pathAPI + pathV1 + routePercentage
	pathSqrt       = pathAPI + pathV1 + routeSqrt
)

// maxBodyBytes is the largest accepted request body (1 MiB).
const maxBodyBytes = 1 << 20

// Struct tags of request types (request.go).
const (
	tagKeyJSON       = "json"
	tagKeyBinding    = "binding"
	jsonTagSeparator = ","
	ruleParamSep     = "="
	jsonTagSkip      = "-"
)

// Binding rules used in request types. omitempty, required, min and max are
// validator built-ins; jsonnumber, finite and integer are registered by newValidator.
const (
	tagOmitEmpty  = "omitempty"
	tagRequired   = "required"
	tagMin        = "min"
	tagMax        = "max"
	tagJSONNumber = "jsonnumber"
	tagFinite     = "finite"
	tagInteger    = "integer"
)

// bitSize makes strconv parse numbers as float64.
const bitSize = 64

// Content types of the documentation endpoints.
const (
	contentTypeHTML = "text/html; charset=utf-8"
	contentTypeYAML = "application/yaml"
)
