package api

import (
	"encoding/json"
	"net/http"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/store"
)

var (
	ErrInvalidSocketFile  = errors.New("[xdb/api] invalid socket file")
	ErrNoStoresConfigured = errors.New("[xdb/api] at least one store must be configured")
)

// ErrorResponse represents a structured error response for the API.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// MapStoreError maps a store error to an HTTP status code and ErrorResponse.
func MapStoreError(err error) (int, *ErrorResponse) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound, &ErrorResponse{
			Code:    "NOT_FOUND",
			Message: err.Error(),
		}
	case errors.Is(err, store.ErrSchemaModeChanged):
		return http.StatusBadRequest, &ErrorResponse{
			Code:    "SCHEMA_MODE_CHANGED",
			Message: err.Error(),
		}
	case errors.Is(err, store.ErrFieldChangeType):
		return http.StatusBadRequest, &ErrorResponse{
			Code:    "FIELD_CHANGE_TYPE",
			Message: err.Error(),
		}
	default:
		return http.StatusInternalServerError, &ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		}
	}
}

// StoreErrorHandler implements xapi.ErrorHandler for mapping store errors to HTTP responses.
type StoreErrorHandler struct{}

// HandleError writes a structured error response based on the error type.
func (h *StoreErrorHandler) HandleError(w http.ResponseWriter, err error) {
	statusCode, errResp := MapStoreError(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errResp) //nolint:errchkjson
}
