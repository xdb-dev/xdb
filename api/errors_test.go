package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/store"
)

func TestErrorResponse_JSONMarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response api.ErrorResponse
		expected map[string]string
	}{
		{
			name: "basic error response",
			response: api.ErrorResponse{
				Code:    "NOT_FOUND",
				Message: "resource not found",
			},
			expected: map[string]string{
				"code":    "NOT_FOUND",
				"message": "resource not found",
			},
		},
		{
			name: "schema mode changed error",
			response: api.ErrorResponse{
				Code:    "SCHEMA_MODE_CHANGED",
				Message: "cannot change schema mode",
			},
			expected: map[string]string{
				"code":    "SCHEMA_MODE_CHANGED",
				"message": "cannot change schema mode",
			},
		},
		{
			name: "field type change error",
			response: api.ErrorResponse{
				Code:    "FIELD_CHANGE_TYPE",
				Message: "cannot change field type",
			},
			expected: map[string]string{
				"code":    "FIELD_CHANGE_TYPE",
				"message": "cannot change field type",
			},
		},
		{
			name: "empty error response",
			response: api.ErrorResponse{
				Code:    "",
				Message: "",
			},
			expected: map[string]string{
				"code":    "",
				"message": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)

			var result map[string]string
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			assert.Equal(t, tt.expected["code"], result["code"])
			assert.Equal(t, tt.expected["message"], result["message"])
		})
	}
}

func TestErrorResponse_JSONUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		jsonInput    string
		expectedCode string
		expectedMsg  string
		expectError  bool
	}{
		{
			name:         "valid error response",
			jsonInput:    `{"code":"NOT_FOUND","message":"resource not found"}`,
			expectedCode: "NOT_FOUND",
			expectedMsg:  "resource not found",
			expectError:  false,
		},
		{
			name:         "schema mode changed error",
			jsonInput:    `{"code":"SCHEMA_MODE_CHANGED","message":"cannot change schema mode"}`,
			expectedCode: "SCHEMA_MODE_CHANGED",
			expectedMsg:  "cannot change schema mode",
			expectError:  false,
		},
		{
			name:         "field type change error",
			jsonInput:    `{"code":"FIELD_CHANGE_TYPE","message":"cannot change field type"}`,
			expectedCode: "FIELD_CHANGE_TYPE",
			expectedMsg:  "cannot change field type",
			expectError:  false,
		},
		{
			name:         "empty json object",
			jsonInput:    `{}`,
			expectedCode: "",
			expectedMsg:  "",
			expectError:  false,
		},
		{
			name:         "extra fields ignored",
			jsonInput:    `{"code":"ERROR","message":"test","extra":"ignored"}`,
			expectedCode: "ERROR",
			expectedMsg:  "test",
			expectError:  false,
		},
		{
			name:        "invalid json",
			jsonInput:   `{invalid}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response api.ErrorResponse
			err := json.Unmarshal([]byte(tt.jsonInput), &response)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
			assert.Equal(t, tt.expectedMsg, response.Message)
		})
	}
}

func TestErrorResponse_RoundTrip(t *testing.T) {
	t.Parallel()

	original := api.ErrorResponse{
		Code:    "INTERNAL_ERROR",
		Message: "an unexpected error occurred",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded api.ErrorResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Code, decoded.Code)
	assert.Equal(t, original.Message, decoded.Message)
}

func TestMapStoreError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "not found error",
			err:            store.ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "schema mode changed error",
			err:            store.ErrSchemaModeChanged,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "SCHEMA_MODE_CHANGED",
		},
		{
			name:           "field change type error",
			err:            store.ErrFieldChangeType,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "FIELD_CHANGE_TYPE",
		},
		{
			name:           "unknown error",
			err:            errors.New("some unknown error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
		{
			name:           "wrapped not found error",
			err:            errors.Join(errors.New("context"), store.ErrNotFound),
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, errResp := api.MapStoreError(tt.err)

			assert.Equal(t, tt.expectedStatus, status)
			assert.Equal(t, tt.expectedCode, errResp.Code)
			assert.Contains(t, errResp.Message, tt.err.Error())
		})
	}
}

func TestStoreErrorHandler_HandleError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "not found error",
			err:            store.ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "schema mode changed error",
			err:            store.ErrSchemaModeChanged,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "SCHEMA_MODE_CHANGED",
		},
		{
			name:           "field change type error",
			err:            store.ErrFieldChangeType,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "FIELD_CHANGE_TYPE",
		},
		{
			name:           "internal error",
			err:            errors.New("internal failure"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &api.StoreErrorHandler{}
			recorder := httptest.NewRecorder()

			handler.HandleError(recorder, tt.err)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

			var errResp api.ErrorResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &errResp)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedCode, errResp.Code)
			assert.Contains(t, errResp.Message, tt.err.Error())
		})
	}
}
