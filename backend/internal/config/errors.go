package config

import "errors"

var (
	// ErrMissingEnv is returned (wrapped) when a required environment variable is unset or empty.
	ErrMissingEnv = errors.New("missing environment variable")
	// ErrInvalidEnv is returned (wrapped) when an environment variable has an invalid value.
	ErrInvalidEnv = errors.New("invalid environment variable")
)
