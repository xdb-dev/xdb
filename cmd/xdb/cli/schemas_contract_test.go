package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the schemas contract of the CLI: evolution, cascade
// delete, idempotent create, index and unique guardrails, dynamic mode, and
// live describe. The retired e2e suite asserted them through the binary.

func TestSchemas_UpdateEvolvesAndKeepsOldRecords(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://sc.t/notes", "--json", `{"fields":{"title":{"type":"string"}}}`)
	require.Equal(t, ExitOK, code)

	_, _, code = runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://sc.t/notes/n1", "--json", `{"title":"old"}`)
	require.Equal(t, ExitOK, code)

	_, stderr, code := runCLI(t, "--config", cfg, "schemas", "update",
		"--uri", "xdb://sc.t/notes", "--json", `{"fields":{"body":{"type":"string"}}}`)
	require.Equal(t, ExitOK, code, stderr)

	stdout, stderr, code := runCLI(t, "--config", cfg, "records", "get", "--uri", "xdb://sc.t/notes/n1", "-o", "json")
	require.Equal(t, ExitOK, code, stderr)
	assert.Equal(t, "old", jsonDoc(t, stdout)["title"])

	stdout, stderr, code = runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://sc.t/notes/n2", "--json", `{"title":"new","body":"with body"}`, "-o", "json")
	require.Equal(t, ExitOK, code, stderr)
	assert.Equal(t, "with body", jsonDoc(t, stdout)["body"])
}

func TestSchemas_CascadeDeleteSparesSiblings(t *testing.T) {
	cfg := startCLITestDaemon(t)

	for _, name := range []string{"orders", "products"} {
		_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
			"--uri", "xdb://sc.t/"+name, "--json", `{"fields":{"n":{"type":"integer"}}}`)
		require.Equal(t, ExitOK, code)

		_, _, code = runCLI(t, "--config", cfg, "records", "create",
			"--uri", "xdb://sc.t/"+name+"/a", "--json", `{"n":1}`)
		require.Equal(t, ExitOK, code)
	}

	_, stderr, code := runCLI(t, "--config", cfg, "schemas", "delete", "--uri", "xdb://sc.t/orders", "--cascade", "--force")
	require.Equal(t, ExitOK, code, stderr)

	_, stderr, code = runCLI(t, "--config", cfg, "schemas", "get", "--uri", "xdb://sc.t/orders")
	assert.Equal(t, ExitAppError, code)
	assert.Equal(t, CodeNotFound, errorEnvelope(t, stderr)["code"])

	stdout, _, code := runCLI(t, "--config", cfg, "records", "list", "--uri", "xdb://sc.t/products", "-o", "ndjson")
	require.Equal(t, ExitOK, code)
	assert.Equal(t, []string{"a"}, ndjsonIDs(t, stdout), "sibling schema keeps its records")
}

func TestSchemas_CreateIdempotentAndConflict(t *testing.T) {
	cfg := startCLITestDaemon(t)
	def := `{"fields":{"name":{"type":"string","required":true},"qty":{"type":"integer"}}}`

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create", "--uri", "xdb://sc.t/widgets", "--json", def)
	require.Equal(t, ExitOK, code)

	_, stderr, code := runCLI(t, "--config", cfg, "schemas", "create", "--uri", "xdb://sc.t/widgets", "--json", def)
	assert.Equal(t, ExitOK, code, "identical re-create is idempotent: %s", stderr)

	_, stderr, code = runCLI(t, "--config", cfg, "schemas", "create", "--uri", "xdb://sc.t/widgets",
		"--json", `{"fields":{"name":{"type":"string","required":true},"qty":{"type":"integer"},"sku":{"type":"string"}}}`)
	assert.Equal(t, ExitAppError, code)

	env := errorEnvelope(t, stderr)
	assert.Equal(t, CodeConflict, env["code"])
	assert.Equal(t, "schemas", env["resource"])
	assert.Equal(t, "create", env["action"])
}

func TestSchemas_IndexedUniqueGuardrails(t *testing.T) {
	cfg := startCLITestDaemon(t)

	tests := []struct {
		name   string
		action string
		uri    string
		def    string
	}{
		{name: "indexed on array", action: "create", uri: "xdb://sc.t/bad-array", def: `{"fields":{"tags":{"type":"array","elem_type":"string","indexed":true}}}`},
		{name: "unique on json", action: "create", uri: "xdb://sc.t/bad-json", def: `{"fields":{"blob":{"type":"json","unique":true}}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, code := runCLI(t, "--config", cfg, "schemas", tt.action, "--uri", tt.uri, "--json", tt.def)
			assert.NotEqual(t, ExitOK, code)

			env := errorEnvelope(t, stderr)
			assert.Equal(t, CodeSchemaViolation, env["code"])
			assert.Equal(t, "schemas", env["resource"])
			assert.Equal(t, tt.action, env["action"])
		})
	}

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create", "--uri", "xdb://sc.t/plain", "--json", `{"fields":{"sku":{"type":"string"}}}`)
	require.Equal(t, ExitOK, code)

	_, stderr, code := runCLI(t, "--config", cfg, "schemas", "update", "--uri", "xdb://sc.t/plain", "--json", `{"fields":{"sku":{"type":"string","unique":true}}}`)
	assert.NotEqual(t, ExitOK, code)

	env := errorEnvelope(t, stderr)
	assert.Equal(t, CodeSchemaViolation, env["code"])
	assert.Equal(t, "update", env["action"])
}

func TestSchemas_DynamicModeLearnsFields(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://sc.t/ticks", "--json", `{"mode":"dynamic","fields":{"symbol":{"type":"string"}}}`)
	require.Equal(t, ExitOK, code)

	_, stderr, code := runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://sc.t/ticks/t1", "--json", `{"symbol":"RELIANCE","exchange":"NSE"}`)
	require.Equal(t, ExitOK, code, stderr)

	stdout, stderr, code := runCLI(t, "--config", cfg, "schemas", "get", "--uri", "xdb://sc.t/ticks", "-o", "json")
	require.Equal(t, ExitOK, code, stderr)
	assert.Contains(t, stdout, "exchange", "dynamic mode adds the undeclared field to the schema")
}

func TestDescribe_LiveSchemaShowsMarkers(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://sc.t/members",
		"--json", `{"fields":{"email":{"type":"string","unique":true},"status":{"type":"string","indexed":true},"name":{"type":"string"}}}`)
	require.Equal(t, ExitOK, code)

	stdout, stderr, code := runCLI(t, "--config", cfg, "describe", "--uri", "xdb://sc.t/members", "-o", "json")
	require.Equal(t, ExitOK, code, stderr)

	for _, want := range []string{"email", "status", "name", "unique", "indexed"} {
		assert.Contains(t, stdout, want)
	}

	assert.NotContains(t, stderr, "error", stderr)
}
