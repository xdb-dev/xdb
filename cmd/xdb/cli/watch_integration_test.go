package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/rpc/client"
)

// TestWatch_StreamsChangesOverSocket exercises the full watch chain:
// CLI-side client.Stream over the unix socket, SSE framing, the daemon
// event bus, and service publications.
func TestWatch_StreamsChangesOverSocket(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://watch.t/items",
		"--json", `{"fields":{"name":{"type":"string"}}}`)
	require.Equal(t, 0, code)

	c := client.New(filepath.Join(filepath.Dir(cfg), "test.sock"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	frames := make(chan client.StreamFrame, 16)
	done := make(chan error, 1)
	go func() {
		done <- c.Stream(ctx, "watch", &api.WatchRequest{URI: "xdb://watch.t"}, func(f client.StreamFrame) error {
			frames <- f
			return nil
		})
	}()

	// Ready must arrive before any mutation is made.
	select {
	case f := <-frames:
		require.Equal(t, "ready", f.Event)
	case <-time.After(2 * time.Second):
		t.Fatal("no ready frame")
	}

	_, _, code = runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://watch.t/items/i1",
		"--json", `{"name":"first"}`)
	require.Equal(t, 0, code)

	select {
	case f := <-frames:
		require.Equal(t, "event", f.Event)

		var e api.WatchEvent
		require.NoError(t, json.Unmarshal(f.Data, &e))
		assert.Equal(t, "record.create", e.Type)
		assert.Equal(t, "xdb://watch.t/items/i1", e.URI)
		assert.False(t, e.TS.IsZero())
	case <-time.After(2 * time.Second):
		t.Fatal("no event frame after create")
	}

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err, "canceled stream must end cleanly")
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not end on cancel")
	}
}
