package api

import (
	"context"
	"net/http"
)

// Validator is an optional interface for request types.
// If a request implements Validator, it is called after JSON decoding
// and extraction but before the handler.
type Validator interface {
	Validate(ctx context.Context) error
}

// Extracter is an optional interface for request types.
// If a request implements Extracter, it is called after JSON decoding
// to extract non-body data from the HTTP request (headers, path params, query strings).
type Extracter interface {
	Extract(r *http.Request) error
}
