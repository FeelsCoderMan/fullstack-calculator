package handler

// URL paths served outside the versioned API.
const (
	PathHealth      = "/health"
	PathSwaggerUI   = "/swagger/"
	PathSwaggerSpec = "/swagger/openapi.yaml"
)

// Content types of the documentation endpoints.
const (
	contentTypeHTML = "text/html; charset=utf-8"
	contentTypeYAML = "application/yaml"
)

// Log messages.
const (
	msgRequestCompleted = "request completed"
)
