package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
)

func TestTransport_TCPConnection(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	config := &api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}

	builder := api.NewClientBuilder(config).
		WithSchemaStore()

	client, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, client)

	err = client.Ping(context.Background())
	assert.NoError(t, err, "Client should connect successfully via TCP")
}

func TestTransport_UnixSocketConnection(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join("/tmp", fmt.Sprintf("xdb-test-%d.sock", os.Getpid()))
	defer os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}),
	}

	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Close()

	config := &api.ClientConfig{
		SocketPath: socketPath,
	}

	builder := api.NewClientBuilder(config).
		WithSchemaStore()

	client, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, client)

	err = client.Ping(context.Background())
	assert.NoError(t, err, "Client should connect successfully via Unix socket")
}

func TestTransport_RequestTimeoutEnforcement(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	config := &api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 100 * time.Millisecond,
	}

	builder := api.NewClientBuilder(config).
		WithSchemaStore()

	client, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, client)

	err = client.Ping(context.Background())
	require.Error(t, err, "Request should timeout")
	assert.True(t, os.IsTimeout(err) || isTimeoutError(err), "Error should be a timeout error")
}

func TestTransport_ContextCancellation(t *testing.T) {
	t.Parallel()

	requestStarted := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		time.Sleep(5 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	config := &api.ClientConfig{
		Addr:    server.Listener.Addr().String(),
		Timeout: 30 * time.Second,
	}

	builder := api.NewClientBuilder(config).
		WithSchemaStore()

	client, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, client)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- client.Ping(ctx)
	}()

	<-requestStarted
	cancel()

	select {
	case err := <-done:
		require.Error(t, err, "Request should be cancelled")
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("Request did not return after context cancellation")
	}
}

func TestTransport_ConnectionRefused(t *testing.T) {
	t.Parallel()

	config := &api.ClientConfig{
		Addr: "localhost:59999",
	}

	builder := api.NewClientBuilder(config).
		WithSchemaStore()

	client, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, client)

	err = client.Ping(context.Background())
	require.Error(t, err, "Connection should be refused")
}

func TestTransport_UnixSocketNotFound(t *testing.T) {
	t.Parallel()

	config := &api.ClientConfig{
		SocketPath: "/tmp/nonexistent-socket-path-12345.sock",
	}

	builder := api.NewClientBuilder(config).
		WithSchemaStore()

	client, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, client)

	err = client.Ping(context.Background())
	require.Error(t, err, "Connection should fail for nonexistent socket")
}

func TestTransport_CustomHTTPClient(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	config := &api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}

	builder := api.NewClientBuilder(config).
		WithHTTPClient(customClient).
		WithSchemaStore()

	client, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, client)

	err = client.Ping(context.Background())
	assert.NoError(t, err, "Client should work with custom HTTP client")
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	type timeoutError interface {
		Timeout() bool
	}

	if te, ok := err.(timeoutError); ok {
		return te.Timeout()
	}

	return false
}
