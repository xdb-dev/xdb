package api

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ErrorHandler handles errors produced during endpoint processing.
type ErrorHandler interface {
	HandleError(w http.ResponseWriter, err error)
}

// ErrorHandlerFunc is a function adapter for [ErrorHandler].
type ErrorHandlerFunc func(w http.ResponseWriter, err error)

// HandleError calls the function.
func (f ErrorHandlerFunc) HandleError(w http.ResponseWriter, err error) {
	f(w, err)
}

// defaultErrorHandler writes a JSON error response with an appropriate
// HTTP status code based on the error type.
type defaultErrorHandler struct{}

// HandleError writes a JSON error response.
func (h *defaultErrorHandler) HandleError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if isJSONError(err) {
		status = http.StatusBadRequest
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}

	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		http.Error(w, encErr.Error(), http.StatusInternalServerError)
	}
}

// isJSONError reports whether err is, or wraps, a decoding error from
// [encoding/json]. Those are caused by a malformed request body, so they
// map to 400 rather than 500.
func isJSONError(err error) bool {
	var (
		syntaxErr    *json.SyntaxError
		typeErr      *json.UnmarshalTypeError
		unmarshalErr *json.InvalidUnmarshalError
	)

	return errors.As(err, &syntaxErr) ||
		errors.As(err, &typeErr) ||
		errors.As(err, &unmarshalErr)
}
