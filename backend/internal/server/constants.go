package server

import "time"

// Log messages.
const (
	msgServerListening    = "server listening"
	msgFileLoggingEnabled = "file logging enabled; development only"
	msgShutdownStarted    = "shutdown signal received, draining connections"
	msgServerStopped      = "server stopped"
)

// HTTP server settings.
const (
	network           = "tcp"
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
)
