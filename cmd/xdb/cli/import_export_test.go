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
	t.Chdir(dir) // --file paths must live under the working directory
	file := filepath.Join(dir, "records.ndjson")
	// Line 2 omits the required "name" field.
	data := "{\"_id\":\"a\",\"name\":\"widget\"}\n" +
		"{\"_id\":\"b\",\"other\":\"no name here\"}\n"
	require.NoError(t, os.WriteFile(file, []byte(data), 0o600))

	stdout, stderr, code := runCLI(t, "--config", configPath,
		"import",
		"--uri", "xdb://ns/items",
		"--file", "records.ndjson",
	)

	assert.NotEqual(t, ExitOK, code)
	assert.Contains(t, stdout, `"failed": 1`, "summary must land on stdout")

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

	t.Chdir(dir) // --file paths must live under the working directory
	file := filepath.Join(dir, "records.ndjson")
	require.NoError(t, os.WriteFile(file, []byte(`{"_id":"a","name":"widget"}`+"\n"), 0o600))

	_, stderr, code := runCLI(t, "--config", configPath,
		"import",
		"--uri", "xdb://ns/items",
		"--file", "records.ndjson",
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

func TestExport_Modes(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://exp.t/items",
		"--json", `{"fields":{"name":{"type":"string"}}}`)
	require.Equal(t, 0, code)

	for _, id := range []string{"a", "b", "c"} {
		_, _, code := runCLI(t, "--config", cfg, "records", "create",
			"--uri", "xdb://exp.t/items/"+id,
			"--json", `{"name":"`+id+`"}`, "--quiet")
		require.Equal(t, 0, code)
	}

	t.Run("default ndjson", func(t *testing.T) {
		stdout, _, code := runCLI(t, "--config", cfg, "export", "--uri", "xdb://exp.t/items")
		require.Equal(t, 0, code)
		assert.Len(t, strings.Split(strings.TrimSpace(stdout), "\n"), 3)
	})

	t.Run("json renders an array", func(t *testing.T) {
		stdout, _, code := runCLI(t, "--config", cfg, "export", "--uri", "xdb://exp.t/items", "-o", "json")
		require.Equal(t, 0, code)

		var items []map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &items))
		assert.Len(t, items, 3)
	})

	t.Run("table format is rejected", func(t *testing.T) {
		_, stderr, code := runCLI(t, "--config", cfg, "export", "--uri", "xdb://exp.t/items", "-o", "table")
		assert.Equal(t, 3, code)
		assert.Contains(t, stderr, "INVALID_ARGUMENT")
	})

	t.Run("record URI exports a single record", func(t *testing.T) {
		stdout, _, code := runCLI(t, "--config", cfg, "export", "--uri", "xdb://exp.t/items/a")
		require.Equal(t, 0, code)

		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		require.Len(t, lines, 1)
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(lines[0]), &m))
		assert.Equal(t, "a", m["_id"])
	})

	t.Run("namespace URI is rejected", func(t *testing.T) {
		_, stderr, code := runCLI(t, "--config", cfg, "export", "--uri", "xdb://exp.t")
		assert.Equal(t, 3, code)
		assert.Contains(t, stderr, "schema URI")
	})

	t.Run("missing schema is NOT_FOUND", func(t *testing.T) {
		_, stderr, code := runCLI(t, "--config", cfg, "export", "--uri", "xdb://exp.t/nope")
		assert.Equal(t, 1, code)
		assert.Contains(t, stderr, "NOT_FOUND")
	})
}

func TestImport_Accounting(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://imp.t/items",
		"--json", `{"fields":{"name":{"type":"string","required":true}}}`)
	require.Equal(t, 0, code)

	t.Run("summary counts imports", func(t *testing.T) {
		stdout, _, code := runCLIStdin(t,
			"{\"_id\":\"i1\",\"name\":\"one\"}\n\n{\"_id\":\"i2\",\"name\":\"two\"}\n",
			"--config", cfg, "import", "--uri", "xdb://imp.t/items")
		require.Equal(t, 0, code)

		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
		assert.Equal(t, float64(2), doc["imported"])
		assert.Equal(t, float64(0), doc["failed"])
	})

	t.Run("create-only counts conflicts as skipped", func(t *testing.T) {
		stdout, _, code := runCLIStdin(t,
			"{\"_id\":\"i1\",\"name\":\"DIFFERENT\"}\n{\"_id\":\"i3\",\"name\":\"three\"}\n",
			"--config", cfg, "import", "--uri", "xdb://imp.t/items", "--create-only")
		require.Equal(t, 0, code)

		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
		assert.Equal(t, float64(1), doc["imported"])
		assert.Equal(t, float64(1), doc["skipped"])

		// The divergent import must not have overwritten local data.
		out, _, code := runCLI(t, "--config", cfg, "records", "get", "--uri", "xdb://imp.t/items/i1", "-o", "json")
		require.Equal(t, 0, code)
		assert.Contains(t, out, `"one"`)
	})

	t.Run("failure reports line and summary with progress", func(t *testing.T) {
		stdout, stderr, code := runCLIStdin(t,
			"{\"_id\":\"j1\",\"name\":\"ok\"}\n\n{\"_id\":\"j2\"}\n{\"_id\":\"j3\",\"name\":\"never\"}\n",
			"--config", cfg, "import", "--uri", "xdb://imp.t/items")
		require.NotEqual(t, 0, code)
		assert.Contains(t, stderr, "line 3", "physical line numbers count blank lines")

		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
		assert.Equal(t, float64(1), doc["imported"])
		assert.Equal(t, float64(1), doc["failed"])
		assert.Equal(t, float64(3), doc["first_error_line"])
	})
}
