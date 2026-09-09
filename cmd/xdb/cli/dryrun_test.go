package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDryRun_CreateValidatesWithoutWriting(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://dry.t/items",
		"--json", `{"fields":{"name":{"type":"string"}}}`)
	require.Equal(t, 0, code)

	stdout, stderr, code := runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://dry.t/items/i1",
		"--json", `{"name":"x"}`,
		"--dry-run", "-o", "json")
	require.Equal(t, 0, code, "stderr: %s", stderr)

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	assert.Equal(t, true, doc["dry_run"])
	assert.Equal(t, true, doc["valid"])
	assert.Equal(t, "create", doc["would"])

	_, _, code = runCLI(t, "--config", cfg, "records", "get", "--uri", "xdb://dry.t/items/i1")
	assert.Equal(t, 1, code, "dry-run create must not persist the record")
}

func TestDryRun_DeletePreservesRecord(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://dry.t/items",
		"--json", `{"fields":{"name":{"type":"string"}}}`)
	require.Equal(t, 0, code)

	_, _, code = runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://dry.t/items/keep",
		"--json", `{"name":"x"}`)
	require.Equal(t, 0, code)

	stdout, _, code := runCLI(t, "--config", cfg, "records", "delete",
		"--uri", "xdb://dry.t/items/keep",
		"--force", "--dry-run", "-o", "json")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, `"would"`)

	_, _, code = runCLI(t, "--config", cfg, "records", "get", "--uri", "xdb://dry.t/items/keep")
	assert.Equal(t, 0, code, "dry-run delete must preserve the record")
}

func TestDryRun_SchemaUpdateFailsClosed(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, stderr, code := runCLI(t, "--config", cfg, "schemas", "update",
		"--uri", "xdb://dry.t/items",
		"--json", `{"fields":{"name":{"type":"string"}}}`,
		"--dry-run")
	assert.Equal(t, 3, code)
	assert.Contains(t, stderr, "not yet supported")
}

func TestRecordsList_QueryFlagRemoved(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, stderr, code := runCLI(t, "--config", cfg, "records", "list",
		"--uri", "xdb://dry.t/items", "--query", "{}")
	assert.Equal(t, 3, code)
	assert.Contains(t, stderr, "INVALID_ARGUMENT", "stderr: %s", stderr)
}

func TestSchemasDelete_RequiresForce(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://dry.t/guarded",
		"--json", `{"fields":{"name":{"type":"string"}}}`)
	require.Equal(t, 0, code)

	_, stderr, code := runCLI(t, "--config", cfg, "schemas", "delete",
		"--uri", "xdb://dry.t/guarded")
	assert.Equal(t, 3, code)
	assert.Contains(t, stderr, "force")

	_, _, code = runCLI(t, "--config", cfg, "schemas", "get", "--uri", "xdb://dry.t/guarded")
	assert.Equal(t, 0, code, "schema must survive an unforced delete")

	_, _, code = runCLI(t, "--config", cfg, "schemas", "delete",
		"--uri", "xdb://dry.t/guarded", "--force")
	assert.Equal(t, 0, code)
}
