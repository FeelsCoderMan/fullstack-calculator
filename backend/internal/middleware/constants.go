package middleware

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
	// contextKeyFailure holds the Failure recorded with RecordFailure, read
	// once by RequestLogger when it writes the request record.
	contextKeyFailure = "middleware.failure"
)

// Log messages.
const (
	msgRequestCompleted = "request completed"
)
