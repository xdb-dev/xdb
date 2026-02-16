package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/client"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
)

func TestConfig_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      client.Config
		expectError bool
	}{
		{
			name: "valid TCP address",
			config: client.Config{
				Addr: "localhost:8080",
			},
			expectError: false,
		},
		{
			name: "valid TCP address with IP",
			config: client.Config{
				Addr: "127.0.0.1:8080",
			},
			expectError: false,
		},
		{
			name:        "empty config",
			config:      client.Config{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_UnixSocketPreference(t *testing.T) {
	t.Parallel()

	config := client.Config{
		Addr:       "localhost:8080",
		SocketPath: "/tmp/xdb.sock",
	}

	addr := config.EffectiveAddress()

	assert.Equal(t, "/tmp/xdb.sock", addr, "Unix socket should be preferred over TCP when both are configured")
	assert.True(t, config.UsesUnixSocket(), "UsesUnixSocket should return true when socket path is set")
}

func TestConfig_TimeoutDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		config          client.Config
		expectedTimeout time.Duration
	}{
		{
			name: "zero timeout defaults to 30s",
			config: client.Config{
				Addr:    "localhost:8080",
				Timeout: 0,
			},
			expectedTimeout: 30 * time.Second,
		},
		{
			name: "custom timeout is preserved",
			config: client.Config{
				Addr:    "localhost:8080",
				Timeout: 10 * time.Second,
			},
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			timeout := tt.config.EffectiveTimeout()

			assert.Equal(t, tt.expectedTimeout, timeout)
		})
	}
}

func TestBuilder_RequiresAtLeastOneStore(t *testing.T) {
	t.Parallel()

	config := &client.Config{
		Addr: "localhost:8080",
	}

	builder := client.NewBuilder(config)
	c, err := builder.Build()

	require.Error(t, err)
	assert.ErrorIs(t, err, client.ErrNoStoresConfigured)
	assert.Nil(t, c)
}

func TestBuilder_RequiresAddressConfiguration(t *testing.T) {
	t.Parallel()

	config := &client.Config{}

	builder := client.NewBuilder(config).
		WithSchemaStore()

	c, err := builder.Build()

	require.Error(t, err)
	assert.ErrorIs(t, err, client.ErrNoAddressConfigured)
	assert.Nil(t, c)
}

func TestBuilder_WithSchemaStoreOnly(t *testing.T) {
	t.Parallel()

	config := &client.Config{
		Addr: "localhost:8080",
	}

	builder := client.NewBuilder(config).
		WithSchemaStore()

	c, err := builder.Build()

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.NotNil(t, c.Schemas(), "SchemaStore should be available")
}

func TestBuilder_WithMultipleStores(t *testing.T) {
	t.Parallel()

	config := &client.Config{
		Addr: "localhost:8080",
	}

	builder := client.NewBuilder(config).
		WithSchemaStore().
		WithTupleStore().
		WithRecordStore()

	c, err := builder.Build()

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.NotNil(t, c.Schemas(), "SchemaStore should be available")
	assert.NotNil(t, c.Tuples(), "TupleStore should be available")
	assert.NotNil(t, c.Records(), "RecordStore should be available")
}

func TestBuilder_WithTupleStoreOnly(t *testing.T) {
	t.Parallel()

	config := &client.Config{
		Addr: "localhost:8080",
	}

	builder := client.NewBuilder(config).
		WithTupleStore()

	c, err := builder.Build()

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.NotNil(t, c.Tuples(), "TupleStore should be available")
}

func TestBuilder_WithRecordStoreOnly(t *testing.T) {
	t.Parallel()

	config := &client.Config{
		Addr: "localhost:8080",
	}

	builder := client.NewBuilder(config).
		WithRecordStore()

	c, err := builder.Build()

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.NotNil(t, c.Records(), "RecordStore should be available")
}

func TestBuilder_WithUnixSocket(t *testing.T) {
	t.Parallel()

	config := &client.Config{
		SocketPath: "/tmp/xdb.sock",
	}

	builder := client.NewBuilder(config).
		WithSchemaStore()

	c, err := builder.Build()

	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestBuilder_NilConfig(t *testing.T) {
	t.Parallel()

	builder := client.NewBuilder(nil)
	c, err := builder.Build()

	require.Error(t, err)
	assert.Nil(t, c)
}

func TestClient_ErrorMapping_NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		resp := api.ErrorResponse{
			Code:    "NOT_FOUND",
			Message: "resource not found",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithRecordStore().Build()
	require.NoError(t, err)

	ctx := context.Background()
	uri := core.MustParseURI("xdb://test/example/123")

	_, _, err = c.GetRecords(ctx, []*core.URI{uri})

	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrNotFound), "expected error to wrap store.ErrNotFound, got: %v", err)
}

func TestClient_ErrorMapping_SchemaModeChanged(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		resp := api.ErrorResponse{
			Code:    "SCHEMA_MODE_CHANGED",
			Message: "cannot change schema mode",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	ctx := context.Background()
	uri := core.MustParseURI("xdb://test/example")

	err = c.PutSchema(ctx, uri, nil)

	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrSchemaModeChanged), "expected error to wrap store.ErrSchemaModeChanged, got: %v", err)
}

func TestClient_ErrorMapping_FieldChangeType(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		resp := api.ErrorResponse{
			Code:    "FIELD_CHANGE_TYPE",
			Message: "cannot change field type",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	ctx := context.Background()
	uri := core.MustParseURI("xdb://test/example")

	err = c.PutSchema(ctx, uri, nil)

	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrFieldChangeType), "expected error to wrap store.ErrFieldChangeType, got: %v", err)
}

func TestClient_ErrorMapping_GenericHTTPError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		errorCode  string
		errorMsg   string
	}{
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			errorCode:  "INTERNAL_ERROR",
			errorMsg:   "internal server error",
		},
		{
			name:       "service unavailable",
			statusCode: http.StatusServiceUnavailable,
			errorCode:  "SERVICE_UNAVAILABLE",
			errorMsg:   "service temporarily unavailable",
		},
		{
			name:       "bad gateway",
			statusCode: http.StatusBadGateway,
			errorCode:  "BAD_GATEWAY",
			errorMsg:   "bad gateway",
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			errorCode:  "UNAUTHORIZED",
			errorMsg:   "unauthorized access",
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			errorCode:  "FORBIDDEN",
			errorMsg:   "access forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				resp := api.ErrorResponse{
					Code:    tt.errorCode,
					Message: tt.errorMsg,
				}
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			c, err := client.NewBuilder(&client.Config{
				Addr: server.Listener.Addr().String(),
			}).WithRecordStore().Build()
			require.NoError(t, err)

			ctx := context.Background()
			uri := core.MustParseURI("xdb://test/example/123")

			_, _, err = c.GetRecords(ctx, []*core.URI{uri})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestClient_ErrorMapping_NetworkError(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	listener.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: addr,
	}).WithRecordStore().Build()
	require.NoError(t, err)

	ctx := context.Background()
	uri := core.MustParseURI("xdb://test/example/123")

	_, _, err = c.GetRecords(ctx, []*core.URI{uri})

	require.Error(t, err)

	var netErr net.Error
	isNetError := errors.As(err, &netErr) || isConnectionError(err)
	assert.True(t, isNetError, "expected network/connection error, got: %v", err)
}

func TestClient_ErrorMapping_ContextCanceled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithRecordStore().Build()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	uri := core.MustParseURI("xdb://test/example/123")

	_, _, err = c.GetRecords(ctx, []*core.URI{uri})

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled), "expected context.Canceled error, got: %v", err)
}

func TestClient_ErrorMapping_MalformedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithRecordStore().Build()
	require.NoError(t, err)

	ctx := context.Background()
	uri := core.MustParseURI("xdb://test/example/123")

	_, _, err = c.GetRecords(ctx, []*core.URI{uri})

	require.Error(t, err)
}

func TestClient_ErrorMapping_EmptyErrorResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithRecordStore().Build()
	require.NoError(t, err)

	ctx := context.Background()
	uri := core.MustParseURI("xdb://test/example/123")

	_, _, err = c.GetRecords(ctx, []*core.URI{uri})

	require.Error(t, err)
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsSubstr(errStr, "connection refused") ||
		containsSubstr(errStr, "no such host") ||
		containsSubstr(errStr, "dial tcp") ||
		containsSubstr(errStr, "connect:")
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
