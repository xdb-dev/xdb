package client_test

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/rpc"
	"github.com/xdb-dev/xdb/rpc/client"
)

// echoRequest is a test request type.
type echoRequest struct {
	Message string `json:"message"`
}

// echoResponse is a test response type.
type echoResponse struct {
	Reply string `json:"reply"`
}

// echo is a test handler that echoes back the message.
func echo(_ context.Context, req *echoRequest) (*echoResponse, error) {
	return &echoResponse{Reply: "echo: " + req.Message}, nil
}

// startTestServer starts a JSON-RPC server on a temp Unix socket and returns
// the socket path. The server is shut down when the test completes.
func startTestServer(t *testing.T, r *rpc.Router) string {
	t.Helper()

	dir := t.TempDir()
	sock := filepath.Join(dir, "test.sock")

	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)

	srv := &http.Server{Handler: r}

	go func() { _ = srv.Serve(ln) }()

	t.Cleanup(func() { _ = srv.Close() })

	return sock
}

func TestClient_Call(t *testing.T) {
	r := rpc.NewRouter()
	rpc.RegisterHandler(r, "test.echo", echo)

	sock := startTestServer(t, r)

	c := client.New(sock)

	t.Run("successful call", func(t *testing.T) {
		var resp echoResponse
		err := c.Call(context.Background(), "test.echo", &echoRequest{Message: "hello"}, &resp)
		require.NoError(t, err)
		assert.Equal(t, "echo: hello", resp.Reply)
	})

	t.Run("method not found", func(t *testing.T) {
		var resp echoResponse
		err := c.Call(context.Background(), "test.missing", &echoRequest{}, &resp)
		require.Error(t, err)

		var rpcErr *rpc.Error
		require.ErrorAs(t, err, &rpcErr)
		assert.Equal(t, rpc.CodeMethodNotFound, rpcErr.Code)
	})

	t.Run("nil params", func(t *testing.T) {
		var resp echoResponse
		err := c.Call(context.Background(), "test.echo", nil, &resp)
		require.NoError(t, err)
		assert.Equal(t, "echo: ", resp.Reply)
	})
}

func TestClient_Call_ConnectionError(t *testing.T) {
	sock := filepath.Join(os.TempDir(), "nonexistent-xdb-test.sock")

	c := client.New(sock)

	var resp echoResponse
	err := c.Call(context.Background(), "test.echo", &echoRequest{}, &resp)
	require.Error(t, err)
}
