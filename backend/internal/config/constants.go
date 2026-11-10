package config

// Environment variable names. Every one is required: there are no defaults,
// so a missing or empty value fails startup.
const (
	EnvPort            = "PORT"
	EnvShutdownTimeout = "SHUTDOWN_TIMEOUT"
	EnvLogLevel        = "LOG_LEVEL"
	EnvLogToFile       = "LOG_TO_FILE"
	EnvLogDir          = "LOG_DIR"
)

const (
	minPort = 0 // 0 = OS-assigned port; accepted deliberately for tests, not for deployments
	maxPort = 65535

	addrFormat = ":%d"
)
