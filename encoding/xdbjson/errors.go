package xdbjson

import "errors"

var (
	// ErrNilRecord is returned when a nil record is passed to encoder.
	ErrNilRecord = errors.New("record cannot be nil")

	// ErrInvalidJSON is returned when JSON parsing fails.
	ErrInvalidJSON = errors.New("invalid JSON")

	// ErrMissingID is returned when the ID field is not found in JSON.
	ErrMissingID = errors.New("missing ID field in JSON")

	// ErrEmptyID is returned when the ID field is empty.
	ErrEmptyID = errors.New("ID cannot be empty")

	// ErrMissingNamespace is returned when namespace cannot be determined.
	ErrMissingNamespace = errors.New("namespace not found in JSON and not specified in Options")

	// ErrMissingSchema is returned when schema cannot be determined.
	ErrMissingSchema = errors.New("schema not found in JSON and not specified in Options")
)
