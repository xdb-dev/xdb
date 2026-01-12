package api

import "github.com/gojekfarm/xtools/errors"

var (
	// ErrInvalidSocketFile is returned when a socket file is invalid.
	ErrInvalidSocketFile = errors.New("[xdb/api] invalid socket file")

	// ErrNoStoresConfigured is returned when no stores are configured.
	ErrNoStoresConfigured = errors.New("[xdb/api] at least one store must be configured")
)
