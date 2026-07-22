package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImportRecords_SchemaViolationMidImport guards the import_export.go
// error-wrapping fix: a schema-violating line partway through an import must
// render a single SCHEMA_VIOLATION envelope naming the offending line, not a
// bare "line N: rpc error ..." string, and must not double-prefix the line
// number.
func TestImportRecords_SchemaViolationMidImport(t *testing.T) {
	configPath := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", configPath,
		"schemas", "create",
		"--uri", "xdb://ns/items",
		"--json", `{"fields":{"name":{"type":"string","required":true}}}`,
	)
	require.Equal(t, ExitOK, code)

	dir := t.TempDir()
	file := filepath.Join(dir, "records.ndjson")
	// Line 2 omits the required "name" field.
	data := "{\"_id\":\"a\",\"name\":\"widget\"}\n" +
		"{\"_id\":\"b\",\"other\":\"no name here\"}\n"
	require.NoError(t, os.WriteFile(file, []byte(data), 0o600))

	stdout, stderr, code := runCLI(t, "--config", configPath,
		"import",
		"--uri", "xdb://ns/items",
		"--file", file,
	)

	assert.NotEqual(t, ExitOK, code)
	assert.Empty(t, stdout)

	var env describeErrEnvelope
	require.NoError(t, json.Unmarshal([]byte(stderr), &env), "stderr: %s", stderr)

	assert.Equal(t, CodeSchemaViolation, env.Code)
	assert.Equal(t, "records", env.Resource)
	assert.Equal(t, "upsert", env.Action)
	assert.Equal(t, 1, strings.Count(env.Message, "line 2"), "message must name line 2 exactly once: %s", env.Message)
}

// TestImportRecords_DaemonDown verifies a dropped daemon connection surfaces
// as CONNECTION_REFUSED (exit 2), not a generic app error, even mid-import.
func TestImportRecords_DaemonDown(t *testing.T) {
	configPath, dir := tempCLIConfig(t)

	file := filepath.Join(dir, "records.ndjson")
	require.NoError(t, os.WriteFile(file, []byte(`{"_id":"a","name":"widget"}`+"\n"), 0o600))

	_, stderr, code := runCLI(t, "--config", configPath,
		"import",
		"--uri", "xdb://ns/items",
		"--file", file,
	)

	assert.Equal(t, ExitConnection, code)
	assert.Contains(t, stderr, "CONNECTION_REFUSED")
}

// TestImportRecords_MissingInput_InvalidArgument verifies the no-input-source
// misuse case renders INVALID_ARGUMENT rather than a bare error.
func TestImportRecords_MissingInput_InvalidArgument(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	_, stderr, code := runCLI(t, "--config", configPath,
		"import",
		"--uri", "xdb://ns/items",
		"--file", "/nonexistent/does-not-exist.ndjson",
	)

	assert.Equal(t, ExitInvalidArgs, code)

	var env describeErrEnvelope
	require.NoError(t, json.Unmarshal([]byte(stderr), &env), "stderr: %s", stderr)
	assert.Equal(t, CodeInvalidArgument, env.Code)
	assert.Equal(t, "records", env.Resource)
	assert.Equal(t, "import", env.Action)
}
