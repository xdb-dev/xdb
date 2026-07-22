package daemon_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/api/catalog"
	"github.com/xdb-dev/xdb/cmd/xdb/daemon"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// TestNewRouter_MatchesCatalog proves the router↔catalog two-way
// completeness: every method the live router serves has a catalog entry
// with identical metadata, and every catalog entry is actually served.
func TestNewRouter_MatchesCatalog(t *testing.T) {
	r := daemon.NewRouter(store.New(xdbmemory.NewDriver()), "test-1.0.0")

	catalogMethods := catalog.Methods()

	for _, name := range r.Methods() {
		routerMeta, ok := r.Meta(name)
		require.True(t, ok, "router claims to serve %s but Meta() returned not-ok", name)

		catalogMeta, ok := catalogMethods[name]
		require.True(t, ok, "router serves %s but catalog.Methods() has no entry for it", name)

		assert.Equal(t, catalogMeta, routerMeta, "meta mismatch for method %s", name)
	}

	for name := range catalogMethods {
		_, ok := r.Meta(name)
		assert.True(t, ok, "catalog defines %s but the router does not serve it", name)
	}
}

func TestDaemon_StartStop(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "xdb.sock")

	d := daemon.New(daemon.Config{
		SocketPath: socketPath,
		Version:    "test-1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start daemon in background.
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Start(ctx, store.New(xdbmemory.NewDriver()))
	}()

	// Wait for socket to appear.
	require.Eventually(t, func() bool {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, 2*time.Second, 50*time.Millisecond, "socket never became available")

	// Make an RPC request through the Unix socket.
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}

	client := &http.Client{Transport: transport}

	body := `{"jsonrpc":"2.0","method":"system.health","id":"1"}`
	resp, err := client.Post(
		"http://localhost/rpc",
		"application/json",
		bytes.NewBufferString(body),
	)
	require.NoError(t, err)

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
	}

	err = json.NewDecoder(resp.Body).Decode(&rpcResp)
	resp.Body.Close()
	require.NoError(t, err)

	var health struct {
		Status string `json:"status"`
	}

	err = json.Unmarshal(rpcResp.Result, &health)
	require.NoError(t, err)
	assert.Equal(t, "ok", health.Status)

	// Close idle connections so Stop doesn't hang.
	transport.CloseIdleConnections()

	// Stop daemon.
	err = d.Stop()
	require.NoError(t, err)

	// Wait for Start to return.
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("daemon did not stop in time")
	}
}

func TestDaemon_ContextCancellation(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "xdb.sock")

	d := daemon.New(daemon.Config{
		SocketPath: socketPath,
		Version:    "test-1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Start(ctx, store.New(xdbmemory.NewDriver()))
	}()

	// Wait for socket to appear.
	require.Eventually(t, func() bool {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, 2*time.Second, 50*time.Millisecond, "socket never became available")

	// Cancel context — this simulates SIGTERM triggering cancel().
	cancel()

	// Start should return promptly.
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("daemon did not stop after context cancellation")
	}
}

func TestDaemon_Status_Stopped(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "xdb.sock")

	d := daemon.New(daemon.Config{
		SocketPath: socketPath,
		Version:    "test-1.0.0",
	})

	status, err := d.Status()
	require.NoError(t, err)
	assert.Equal(t, "stopped", status)
}

func TestDaemon_Status_Running(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "xdb.sock")

	d := daemon.New(daemon.Config{
		SocketPath: socketPath,
		Version:    "test-1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Start(ctx, store.New(xdbmemory.NewDriver()))
	}()

	// Wait for socket to appear.
	require.Eventually(t, func() bool {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, 2*time.Second, 50*time.Millisecond, "socket never became available")

	status, err := d.Status()
	require.NoError(t, err)
	assert.Equal(t, "running", status)

	// Clean up.
	err = d.Stop()
	require.NoError(t, err)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("daemon did not stop in time")
	}
}
