package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
)

func TestClient_Health_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/health", r.URL.Path)

		resp := api.HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now().Format(time.RFC3339),
			ProcessInfo: api.ProcessInfo{
				PID:        12345,
				Uptime:     "1h30m",
				Goroutines: 50,
			},
			StoreHealth: api.StoreHealth{
				Status: "healthy",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 5 * time.Second,
	}).WithHealthStore().Build()
	require.NoError(t, err)

	err = client.Health(context.Background())

	assert.NoError(t, err)
}

func TestClient_Health_UnhealthyStore(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errMsg := "database connection failed"
		resp := api.HealthResponse{
			Status:    "unhealthy",
			Timestamp: time.Now().Format(time.RFC3339),
			ProcessInfo: api.ProcessInfo{
				PID:        12345,
				Uptime:     "1h30m",
				Goroutines: 50,
			},
			StoreHealth: api.StoreHealth{
				Status: "unhealthy",
				Error:  &errMsg,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 5 * time.Second,
	}).WithHealthStore().Build()
	require.NoError(t, err)

	err = client.Health(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
}

func TestClient_Health_StoreNotConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when health store is not configured")
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 5 * time.Second,
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	err = client.Health(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, api.ErrHealthStoreNotConfigured)
}

func TestClient_Health_Timeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)

		resp := api.HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now().Format(time.RFC3339),
			ProcessInfo: api.ProcessInfo{
				PID:        12345,
				Uptime:     "1h30m",
				Goroutines: 50,
			},
			StoreHealth: api.StoreHealth{
				Status: "healthy",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 50 * time.Millisecond,
	}).WithHealthStore().Build()
	require.NoError(t, err)

	err = client.Health(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

func TestClient_Health_ResponseDeserialization(t *testing.T) {
	t.Parallel()

	expectedTimestamp := "2026-01-12T10:30:00Z"
	expectedPID := 54321
	expectedUptime := "2h45m30s"
	expectedGoroutines := 100

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := api.HealthResponse{
			Status:    "healthy",
			Timestamp: expectedTimestamp,
			ProcessInfo: api.ProcessInfo{
				PID:        expectedPID,
				Uptime:     expectedUptime,
				Goroutines: expectedGoroutines,
			},
			StoreHealth: api.StoreHealth{
				Status: "healthy",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 5 * time.Second,
	}).WithHealthStore().Build()
	require.NoError(t, err)

	err = client.Health(context.Background())

	assert.NoError(t, err)
}

func TestClient_Health_UnhealthyWithoutErrorMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := api.HealthResponse{
			Status:    "unhealthy",
			Timestamp: time.Now().Format(time.RFC3339),
			ProcessInfo: api.ProcessInfo{
				PID:        12345,
				Uptime:     "1h30m",
				Goroutines: 50,
			},
			StoreHealth: api.StoreHealth{
				Status: "unhealthy",
				Error:  nil,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 5 * time.Second,
	}).WithHealthStore().Build()
	require.NoError(t, err)

	err = client.Health(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unhealthy")
}

func TestClient_Health_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"INTERNAL_ERROR","message":"internal server error"}`))
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 5 * time.Second,
	}).WithHealthStore().Build()
	require.NoError(t, err)

	err = client.Health(context.Background())

	assert.Error(t, err)
}

func TestClient_Health_InvalidJSONResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 5 * time.Second,
	}).WithHealthStore().Build()
	require.NoError(t, err)

	err = client.Health(context.Background())

	assert.Error(t, err)
}

func TestClient_Health_ContextCancellation(t *testing.T) {
	t.Parallel()

	requestReceived := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestReceived)
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 10 * time.Second,
	}).WithHealthStore().Build()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- client.Health(ctx)
	}()

	<-requestReceived
	cancel()

	err = <-done
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
