// Package config loads configuration from environment variables. Each concern
// has its own loader (LoadServer, LoadLog), so a caller reads and validates
// only the variables it actually uses. Every variable is required; there are
// no defaults.
package config

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// Server configures the HTTP server.
type Server struct {
	Port            int
	ShutdownTimeout time.Duration
}

// Addr returns the listen address for the HTTP server.
func (s Server) Addr() string {
	return fmt.Sprintf(addrFormat, s.Port)
}

// LoadServer reads PORT and SHUTDOWN_TIMEOUT from getenv (typically os.Getenv).
func LoadServer(getenv func(string) string) (Server, error) {
	v, err := require(getenv, EnvPort)
	if err != nil {
		return Server{}, err
	}
	port, err := strconv.Atoi(v)
	if err != nil || port < minPort || port > maxPort {
		return Server{}, fmt.Errorf("%w %s=%q: must be an integer between %d and %d", ErrInvalidEnv, EnvPort, v, minPort, maxPort)
	}

	if v, err = require(getenv, EnvShutdownTimeout); err != nil {
		return Server{}, err
	}
	timeout, err := time.ParseDuration(v)
	if err != nil || timeout <= 0 {
		return Server{}, fmt.Errorf("%w %s=%q: must be a positive duration such as 10s", ErrInvalidEnv, EnvShutdownTimeout, v)
	}

	return Server{Port: port, ShutdownTimeout: timeout}, nil
}

// Log configures structured logging output.
type Log struct {
	Level slog.Level
	// ToFile additionally writes logs to per-category files in Dir. Development only.
	ToFile bool
	Dir    string
}

// LoadLog reads LOG_LEVEL, LOG_TO_FILE and LOG_DIR from getenv (typically os.Getenv).
func LoadLog(getenv func(string) string) (Log, error) {
	var l Log

	v, err := require(getenv, EnvLogLevel)
	if err != nil {
		return Log{}, err
	}
	if err := l.Level.UnmarshalText([]byte(v)); err != nil {
		return Log{}, fmt.Errorf("%w %s=%q: must be debug, info, warn or error, optionally with an offset such as info+2", ErrInvalidEnv, EnvLogLevel, v)
	}

	if v, err = require(getenv, EnvLogToFile); err != nil {
		return Log{}, err
	}
	if l.ToFile, err = strconv.ParseBool(v); err != nil {
		return Log{}, fmt.Errorf("%w %s=%q: must be a boolean such as true or false", ErrInvalidEnv, EnvLogToFile, v)
	}

	if l.Dir, err = require(getenv, EnvLogDir); err != nil {
		return Log{}, err
	}

	return l, nil
}

// require returns the value of the environment variable name, or an error
// wrapping ErrMissingEnv when it is unset or empty.
func require(getenv func(string) string, name string) (string, error) {
	v := getenv(name)
	if v == "" {
		return "", fmt.Errorf("%w %s", ErrMissingEnv, name)
	}
	return v, nil
}
