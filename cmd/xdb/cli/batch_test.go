package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeBatchOps(t *testing.T) {
	t.Run("passes through array unchanged", func(t *testing.T) {
		in := json.RawMessage(`[{"resource":"records","action":"create","uri":"xdb://x/y/z","payload":{"a":1}}]`)
		out, err := normalizeBatchOps(in)
		require.NoError(t, err)
		assert.JSONEq(t, string(in), string(out))
	})

	t.Run("converts ndjson to array", func(t *testing.T) {
		in := json.RawMessage(
			`{"resource":"records","action":"create","uri":"xdb://x/y/a","payload":{"t":"A"}}
{"resource":"records","action":"update","uri":"xdb://x/y/b","payload":{"t":"B"}}`,
		)
		out, err := normalizeBatchOps(in)
		require.NoError(t, err)

		var parsed []map[string]any
		require.NoError(t, json.Unmarshal(out, &parsed))
		assert.Len(t, parsed, 2)
		assert.Equal(t, "records", parsed[0]["resource"])
		assert.Equal(t, "create", parsed[0]["action"])
		assert.Equal(t, "update", parsed[1]["action"])
	})

	t.Run("tolerates leading whitespace", func(t *testing.T) {
		in := json.RawMessage(
			"\n  \t" + `{"resource":"records","action":"get","uri":"xdb://x/y/z"}` + "\n",
		)
		out, err := normalizeBatchOps(in)
		require.NoError(t, err)

		var parsed []map[string]any
		require.NoError(t, json.Unmarshal(out, &parsed))
		assert.Len(t, parsed, 1)
	})

	t.Run("rejects empty", func(t *testing.T) {
		_, err := normalizeBatchOps(json.RawMessage(""))
		assert.Error(t, err)
	})

	t.Run("rejects non-object/array input", func(t *testing.T) {
		_, err := normalizeBatchOps(json.RawMessage(`"not an object"`))
		assert.Error(t, err)
	})

	t.Run("reports ndjson parse error with line number", func(t *testing.T) {
		in := json.RawMessage(
			`{"resource":"records","action":"create"}
{not valid json}`,
		)
		_, err := normalizeBatchOps(in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "operation 2")
	})
}
