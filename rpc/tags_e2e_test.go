package rpc_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/rpc"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// TestSchemaViolationTagsReachData drives a real schema violation from the
// store through MapError and asserts the tags arrive as structured data.
// The unit tests cover the mapping in isolation. This one covers the
// wiring between the layers, where a break produces no failure of its own.
func TestSchemaViolationTagsReachData(t *testing.T) {
	ctx := context.Background()
	st := store.New(xdbmemory.NewDriver())

	schemas := api.NewSchemaService(st)
	_, err := schemas.Create(ctx, &api.CreateSchemaRequest{
		URI:  "xdb://com.example/posts",
		Data: json.RawMessage(`{"mode":"strict","fields":{"count":{"type":"integer"}}}`),
	})
	require.NoError(t, err)

	records := api.NewRecordService(st)
	_, err = records.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://com.example/posts/post-1",
		Data: json.RawMessage(`{"count":"not-an-integer"}`),
	})
	require.Error(t, err)

	rpcErr := rpc.MapError(err)
	assert.Equal(t, rpc.CodeSchemaViolation, rpcErr.Code)

	data, ok := rpcErr.Data.(map[string]string)
	require.True(t, ok, "expected structured Data, got %T (%v)", rpcErr.Data, rpcErr.Data)

	assert.Equal(t, "count", data["field"])
	assert.Equal(t, "INTEGER", data["expected"])
	assert.Equal(t, "STRING", data["got"])
	assert.Equal(t, "decode_failed", data["reason"])

	// The message may still carry the tags as text, and their order there is
	// not stable. Data is the part with a contract.
	assert.True(t, strings.Contains(rpcErr.Message, "count"),
		"message should still name the field for a human: %q", rpcErr.Message)
}
