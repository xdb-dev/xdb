package api

import "net/http"

// StatusCoder is an optional interface for response types.
// If a response implements StatusCoder, the returned status code
// is used instead of the default 200 OK.
type StatusCoder interface {
	StatusCode() int
}

// RawWriter is an optional interface for response types.
// If a response implements RawWriter, it bypasses JSON encoding
// and writes directly to the [http.ResponseWriter].
type RawWriter interface {
	Write(w http.ResponseWriter) error
}
