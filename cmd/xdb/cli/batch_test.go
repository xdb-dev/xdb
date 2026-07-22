package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeBatchOps(t *testing.T) {
	t.Run("parses a JSON array of operations", func(t *testing.T) {
		in := json.RawMessage(`[{"op":"records.create","uri":"xdb://x/y/z","data":{"a":1}}]`)
		ops, err := normalizeBatchOps(in)
		require.NoError(t, err)
		require.Len(t, ops, 1)
		assert.Equal(t, "records.create", ops[0].Op)
		assert.Equal(t, "xdb://x/y/z", ops[0].URI)
		assert.JSONEq(t, `{"a":1}`, string(ops[0].Data))
	})

	t.Run("converts ndjson to operations", func(t *testing.T) {
		in := json.RawMessage(
			`{"op":"records.create","uri":"xdb://x/y/a","data":{"t":"A"}}
{"op":"records.update","uri":"xdb://x/y/b","data":{"t":"B"}}`,
		)
		ops, err := normalizeBatchOps(in)
		require.NoError(t, err)
		require.Len(t, ops, 2)
		assert.Equal(t, "records.create", ops[0].Op)
		assert.Equal(t, "records.update", ops[1].Op)
	})

	t.Run("tolerates leading whitespace", func(t *testing.T) {
		in := json.RawMessage(
			"\n  \t" + `{"op":"records.delete","uri":"xdb://x/y/z"}` + "\n",
		)
		ops, err := normalizeBatchOps(in)
		require.NoError(t, err)
		require.Len(t, ops, 1)
		assert.Equal(t, "records.delete", ops[0].Op)
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
			`{"op":"records.create","uri":"xdb://x/y/a"}
{not valid json}`,
		)
		_, err := normalizeBatchOps(in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "operation 2")
	})
}

func TestBatchExecute_EndToEnd(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://batch.cli/items",
		"--json", `{"fields":{"name":{"type":"string","required":true}}}`)
	require.Equal(t, 0, code)

	t.Run("atomic batch succeeds", func(t *testing.T) {
		stdout, stderr, code := runCLI(t, "--config", cfg, "batch",
			"--json", `[
				{"op":"records.create","uri":"xdb://batch.cli/items/a","data":{"name":"A"}},
				{"op":"records.create","uri":"xdb://batch.cli/items/b","data":{"name":"B"}}
			]`, "-o", "json")
		require.Equal(t, 0, code, "stderr: %s", stderr)

		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &m))
		assert.Equal(t, float64(2), m["succeeded"])
	})

	t.Run("mid-batch violation rolls back", func(t *testing.T) {
		stdout, _, code := runCLI(t, "--config", cfg, "batch",
			"--json", `[
				{"op":"records.create","uri":"xdb://batch.cli/items/c","data":{"name":"C"}},
				{"op":"records.create","uri":"xdb://batch.cli/items/bad","data":{}}
			]`, "-o", "json")
		require.Equal(t, 0, code)

		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &m))
		assert.Equal(t, true, m["rolled_back"])

		_, _, code = runCLI(t, "--config", cfg, "records", "get", "--uri", "xdb://batch.cli/items/c")
		assert.Equal(t, 1, code, "op before the failure must be rolled back")
	})

	t.Run("dry-run batch writes nothing", func(t *testing.T) {
		_, _, code := runCLI(t, "--config", cfg, "batch",
			"--json", `[{"op":"records.create","uri":"xdb://batch.cli/items/dry","data":{"name":"D"}}]`,
			"--dry-run")
		require.Equal(t, 0, code)

		_, _, code = runCLI(t, "--config", cfg, "records", "get", "--uri", "xdb://batch.cli/items/dry")
		assert.Equal(t, 1, code)
	})
}
