package cli_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/cmd/xdb/daemon"
	"github.com/xdb-dev/xdb/rpc/client"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// startTestDaemon starts a daemon with an in-memory store on a temp socket.
func startTestDaemon(t *testing.T) *client.Client {
	t.Helper()

	// Use /tmp directly — t.TempDir() paths are too long for Unix sockets on macOS.
	dir, err := os.MkdirTemp("/tmp", "xdb-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	sock := filepath.Join(dir, "test.sock")
	s := xdbmemory.New()
	router := daemon.NewRouter(s, "test")

	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)

	srv := &http.Server{Handler: router}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	return client.New(sock)
}

func TestIntegration_RecordLifecycle(t *testing.T) {
	c := startTestDaemon(t)
	ctx := context.Background()

	// Create a schema first.
	var schemaResp api.CreateSchemaResponse
	require.NoError(t, c.Call(ctx, "schemas.create", &api.CreateSchemaRequest{
		URI:  "xdb://com.test/posts",
		Data: json.RawMessage(`{"fields":{"title":{"type":"string"}}}`),
	}, &schemaResp))

	// Create a record.
	var createResp api.CreateRecordResponse
	require.NoError(t, c.Call(ctx, "records.create", &api.CreateRecordRequest{
		URI:  "xdb://com.test/posts/post-1",
		Data: json.RawMessage(`{"title":"Hello"}`),
	}, &createResp))

	var m map[string]any
	require.NoError(t, json.Unmarshal(createResp.Data, &m))
	assert.Equal(t, "Hello", m["title"])

	// Get the record.
	var getResp api.GetRecordResponse
	require.NoError(t, c.Call(ctx, "records.get", &api.GetRecordRequest{
		URI: "xdb://com.test/posts/post-1",
	}, &getResp))

	require.NoError(t, json.Unmarshal(getResp.Data, &m))
	assert.Equal(t, "Hello", m["title"])

	// Update the record.
	var updateResp api.UpdateRecordResponse
	require.NoError(t, c.Call(ctx, "records.update", &api.UpdateRecordRequest{
		URI:  "xdb://com.test/posts/post-1",
		Data: json.RawMessage(`{"title":"Updated"}`),
	}, &updateResp))

	require.NoError(t, json.Unmarshal(updateResp.Data, &m))
	assert.Equal(t, "Updated", m["title"])

	// List records.
	var listResp api.ListRecordsResponse
	require.NoError(t, c.Call(ctx, "records.list", &api.ListRecordsRequest{
		URI: "xdb://com.test/posts",
	}, &listResp))
	assert.Equal(t, 1, listResp.Total)
	assert.Len(t, listResp.Items, 1)

	// Delete the record.
	require.NoError(t, c.Call(ctx, "records.delete", &api.DeleteRecordRequest{
		URI: "xdb://com.test/posts/post-1",
	}, nil))

	// Verify deleted.
	require.NoError(t, c.Call(ctx, "records.list", &api.ListRecordsRequest{
		URI: "xdb://com.test/posts",
	}, &listResp))
	assert.Equal(t, 0, listResp.Total)
}

func TestIntegration_Introspect(t *testing.T) {
	c := startTestDaemon(t)
	ctx := context.Background()

	t.Run("list methods", func(t *testing.T) {
		var resp api.ListMethodsResponse
		require.NoError(t, c.Call(ctx, "introspect.methods", &api.ListMethodsRequest{}, &resp))
		assert.NotEmpty(t, resp.Methods)

		// Check that records.create is in the list with a description.
		found := false
		for _, m := range resp.Methods {
			if m.Method == "records.create" {
				found = true
				assert.NotEmpty(t, m.Description)
				assert.True(t, m.Mutating)
			}
		}

		assert.True(t, found, "records.create should be in method list")
	})

	t.Run("describe method", func(t *testing.T) {
		var resp api.DescribeMethodResponse
		require.NoError(t, c.Call(ctx, "introspect.method", &api.DescribeMethodRequest{
			Method: "records.list",
		}, &resp))

		assert.Equal(t, "records.list", resp.Method)
		assert.NotEmpty(t, resp.Description)
		assert.NotEmpty(t, resp.Parameters)
		assert.Contains(t, resp.Parameters, "uri")
		assert.Contains(t, resp.Parameters, "filter")
		assert.NotEmpty(t, resp.Response)
		assert.Contains(t, resp.Response, "items")
	})

	t.Run("list types", func(t *testing.T) {
		var resp api.ListTypesResponse
		require.NoError(t, c.Call(ctx, "introspect.types", &api.ListTypesRequest{}, &resp))
		assert.NotEmpty(t, resp.Types)

		found := false
		for _, t2 := range resp.Types {
			if t2.Type == "Record" {
				found = true
				assert.NotEmpty(t, t2.Description)
			}
		}

		assert.True(t, found, "Record should be in type list")
	})

	t.Run("describe type", func(t *testing.T) {
		var resp api.DescribeTypeResponse
		require.NoError(t, c.Call(ctx, "introspect.type", &api.DescribeTypeRequest{
			Type: "URI",
		}, &resp))

		assert.Equal(t, "URI", resp.Type)
		assert.NotEmpty(t, resp.Description)
	})

	t.Run("unknown method", func(t *testing.T) {
		var resp api.DescribeMethodResponse
		err := c.Call(ctx, "introspect.method", &api.DescribeMethodRequest{
			Method: "nonexistent.method",
		}, &resp)
		require.Error(t, err)
	})
}

func TestIntegration_SystemHealth(t *testing.T) {
	c := startTestDaemon(t)
	ctx := context.Background()

	var resp map[string]string
	require.NoError(t, c.Call(ctx, "system.health", nil, &resp))
	assert.Equal(t, "ok", resp["status"])
}
