package handler

// URL paths served by the API.
const (
	PathHealth = "/health"
)

// HealthStatus is the liveness state reported by the health endpoint.
type HealthStatus string

// Health statuses.
const (
	HealthStatusOK HealthStatus = "ok"
)

// Log messages.
const (
	msgRequestCompleted = "request completed"
)
