package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the records contract of the CLI: exit codes, error
// envelopes, patch and replace, filters, projections, and type fidelity. The
// retired e2e suite asserted them through the binary.

// errorEnvelope parses the JSON error envelope from stderr.
func errorEnvelope(t *testing.T, stderr string) map[string]any {
	t.Helper()

	start := strings.Index(stderr, "{")
	require.GreaterOrEqual(t, start, 0, "no envelope in stderr: %s", stderr)

	var env map[string]any
	require.NoError(t, json.Unmarshal([]byte(stderr[start:]), &env), "stderr: %s", stderr)

	return env
}

func jsonDoc(t *testing.T, stdout string) map[string]any {
	t.Helper()

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc), "stdout: %s", stdout)

	return doc
}

func ndjsonIDs(t *testing.T, stdout string) []string {
	t.Helper()

	var ids []string

	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if line == "" {
			continue
		}

		doc := jsonDoc(t, line)
		ids = append(ids, doc["_id"].(string))
	}

	return ids
}

func seedWidgets(t *testing.T, cfg string) {
	t.Helper()

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://rc.t/widgets",
		"--json", `{"fields":{"name":{"type":"string","required":true},"qty":{"type":"integer"},"active":{"type":"boolean"}}}`)
	require.Equal(t, ExitOK, code)

	_, _, code = runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://rc.t/widgets/w1",
		"--json", `{"name":"sprocket","qty":10,"active":true}`)
	require.Equal(t, ExitOK, code)
}

func TestRecords_DeleteRequiresForce(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedWidgets(t, cfg)

	_, stderr, code := runCLI(t, "--config", cfg, "records", "delete", "--uri", "xdb://rc.t/widgets/w1")
	assert.Equal(t, ExitInvalidArgs, code)
	assert.Contains(t, stderr, "force")

	env := errorEnvelope(t, stderr)
	assert.Equal(t, CodeInvalidArgument, env["code"])
	assert.Equal(t, "records", env["resource"])
	assert.Equal(t, "delete", env["action"])

	_, _, code = runCLI(t, "--config", cfg, "records", "delete", "--uri", "xdb://rc.t/widgets/w1", "--force")
	assert.Equal(t, ExitOK, code)

	_, _, code = runCLI(t, "--config", cfg, "records", "delete", "--uri", "xdb://rc.t/widgets/w1", "--force")
	assert.Equal(t, ExitOK, code, "delete --force is idempotent")

	_, _, code = runCLI(t, "--config", cfg, "records", "get", "--uri", "xdb://rc.t/widgets/w1")
	assert.Equal(t, ExitAppError, code)
}

func TestRecords_UpdatePatchesUpsertReplaces(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedWidgets(t, cfg)

	stdout, stderr, code := runCLI(t, "--config", cfg, "records", "update",
		"--uri", "xdb://rc.t/widgets/w1", "--json", `{"qty":11}`, "-o", "json")
	require.Equal(t, ExitOK, code, stderr)

	doc := jsonDoc(t, stdout)
	assert.Equal(t, "sprocket", doc["name"], "patch keeps untouched fields")
	assert.InDelta(t, 11, doc["qty"], 0)
	assert.Equal(t, true, doc["active"])

	stdout, stderr, code = runCLI(t, "--config", cfg, "records", "upsert",
		"--uri", "xdb://rc.t/widgets/w1", "--json", `{"name":"sprocket","qty":12}`, "-o", "json")
	require.Equal(t, ExitOK, code, stderr)

	doc = jsonDoc(t, stdout)
	assert.InDelta(t, 12, doc["qty"], 0)
	assert.NotContains(t, doc, "active", "upsert replaces the whole record")
}

func TestRecords_CreateIdempotentAndConflict(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedWidgets(t, cfg)

	stdout, stderr, code := runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://rc.t/widgets/w1", "--json", `{"name":"sprocket","qty":10,"active":true}`, "-o", "json")
	require.Equal(t, ExitOK, code, stderr)
	assert.Equal(t, "sprocket", jsonDoc(t, stdout)["name"])

	_, stderr, code = runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://rc.t/widgets/w1", "--json", `{"name":"sprocket","qty":99}`, "-o", "json")
	assert.Equal(t, ExitAppError, code)

	env := errorEnvelope(t, stderr)
	assert.Equal(t, CodeConflict, env["code"])
	assert.Equal(t, "records", env["resource"])
	assert.Equal(t, "create", env["action"])
}

func TestRecords_SchemaViolationEnvelope(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedWidgets(t, cfg)

	tests := []struct {
		name string
		args []string
	}{
		{name: "wrong type", args: []string{"--json", `{"name":"x","qty":"not-a-number"}`}},
		{name: "missing required", args: []string{"--json", `{"qty":1}`}},
		{name: "undeclared field in strict mode", args: []string{"--json", `{"name":"x","color":"red"}`}},
		{name: "dry run still validates", args: []string{"--json", `{"name":"x","qty":"nope"}`, "--dry-run"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"--config", cfg, "records", "create", "--uri", "xdb://rc.t/widgets/bad", "-o", "json"}, tt.args...)

			stdout, stderr, code := runCLI(t, args...)
			assert.Equal(t, ExitAppError, code)
			assert.Empty(t, strings.TrimSpace(stdout))

			env := errorEnvelope(t, stderr)
			assert.Equal(t, CodeSchemaViolation, env["code"])
			assert.Equal(t, "records", env["resource"])
			assert.Equal(t, "create", env["action"])
		})
	}
}

func TestRecords_NotFoundEnvelope(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedWidgets(t, cfg)

	stdout, stderr, code := runCLI(t, "--config", cfg, "records", "get", "--uri", "xdb://rc.t/widgets/ghost", "-o", "json")
	assert.Equal(t, ExitAppError, code)
	assert.Empty(t, strings.TrimSpace(stdout))

	env := errorEnvelope(t, stderr)
	assert.Equal(t, CodeNotFound, env["code"])
	assert.Equal(t, "records", env["resource"])
	assert.Equal(t, "get", env["action"])
	assert.Contains(t, env["hint"], "xdb records list")
}

func TestRecords_QuietSuppressesStdout(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedWidgets(t, cfg)

	stdout, _, code := runCLI(t, "--config", cfg, "records", "get", "--uri", "xdb://rc.t/widgets/w1", "--quiet")
	assert.Equal(t, ExitOK, code)
	assert.Empty(t, strings.TrimSpace(stdout))
}

func TestRecords_ListFilterFieldsOffset(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedWidgets(t, cfg)

	_, _, code := runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://rc.t/widgets/w2", "--json", `{"name":"gear","qty":3,"active":false}`)
	require.Equal(t, ExitOK, code)

	_, _, code = runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://rc.t/widgets/w3", "--json", `{"name":"cog","qty":30,"active":true}`)
	require.Equal(t, ExitOK, code)

	stdout, stderr, code := runCLI(t, "--config", cfg, "records", "list",
		"--uri", "xdb://rc.t/widgets", "--filter", `active == true && qty >= 10`, "--fields", "_id,name", "-o", "ndjson")
	require.Equal(t, ExitOK, code, stderr)
	assert.ElementsMatch(t, []string{"w1", "w3"}, ndjsonIDs(t, stdout))

	first := jsonDoc(t, strings.SplitN(strings.TrimSpace(stdout), "\n", 2)[0])
	assert.Contains(t, first, "name")
	assert.NotContains(t, first, "qty", "--fields projects the record")

	page1, _, code := runCLI(t, "--config", cfg, "records", "list", "--uri", "xdb://rc.t/widgets", "--limit", "2", "-o", "ndjson")
	require.Equal(t, ExitOK, code)
	page2, _, code := runCLI(t, "--config", cfg, "records", "list", "--uri", "xdb://rc.t/widgets", "--limit", "2", "--offset", "2", "-o", "ndjson")
	require.Equal(t, ExitOK, code)

	ids := append(ndjsonIDs(t, page1), ndjsonIDs(t, page2)...)
	assert.ElementsMatch(t, []string{"w1", "w2", "w3"}, ids, "pages are disjoint and complete")
}

func TestRecords_ListFilterErrors(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedWidgets(t, cfg)

	tests := []struct {
		name   string
		filter string
		names  string
	}{
		{name: "syntax error", filter: "name ==", names: ""},
		{name: "unknown field on strict schema", filter: `nosuchfield == "x"`, names: "nosuchfield"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, code := runCLI(t, "--config", cfg, "records", "list", "--uri", "xdb://rc.t/widgets", "--filter", tt.filter)
			assert.Equal(t, ExitInvalidArgs, code)

			env := errorEnvelope(t, stderr)
			assert.Equal(t, CodeInvalidArgument, env["code"])
			assert.Equal(t, "records", env["resource"])
			assert.Equal(t, "list", env["action"])

			if tt.names != "" {
				assert.Contains(t, stderr, tt.names)
			}
		})
	}
}

func TestRecords_NestedPatchPreservesSiblings(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://rc.t/contacts",
		"--json", `{"mode":"flexible","fields":{"name":{"type":"string"},"address.city":{"type":"string"},"address.state":{"type":"string"}}}`)
	require.Equal(t, ExitOK, code)

	stdout, stderr, code := runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://rc.t/contacts/priya",
		"--json", `{"name":"Priya","gstin":"27ABCDE1234F1Z5","address":{"city":"Mumbai","state":"MH"}}`, "-o", "json")
	require.Equal(t, ExitOK, code, stderr)
	assert.Equal(t, "27ABCDE1234F1Z5", jsonDoc(t, stdout)["gstin"], "flexible mode keeps undeclared fields")

	stdout, stderr, code = runCLI(t, "--config", cfg, "records", "update",
		"--uri", "xdb://rc.t/contacts/priya", "--json", `{"address":{"city":"Pune"}}`, "-o", "json")
	require.Equal(t, ExitOK, code, stderr)

	address, ok := jsonDoc(t, stdout)["address"].(map[string]any)
	require.True(t, ok, "stdout: %s", stdout)
	assert.Equal(t, "Pune", address["city"])
	assert.Equal(t, "MH", address["state"], "nested patch keeps sibling keys")
}

func TestRecords_TypeFidelity(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://rc.t/samples",
		"--json", `{"fields":{"s":{"type":"string"},"i":{"type":"integer"},"u":{"type":"unsigned"},"f":{"type":"float"},"b":{"type":"boolean"},"t":{"type":"time"},"by":{"type":"bytes"},"tags":{"type":"array","elem_type":"string"},"prices":{"type":"array","elem_type":"float"}}}`)
	require.Equal(t, ExitOK, code)

	payload := `{"s":"नमस्ते","i":-42,"u":998877665544,"f":0.07,"b":true,"t":"2024-06-01T10:00:00Z","by":"aGVsbG8=","tags":["silk","handloom"],"prices":[12999.5,499.5]}`

	_, stderr, code := runCLI(t, "--config", cfg, "records", "create", "--uri", "xdb://rc.t/samples/r1", "--json", payload)
	require.Equal(t, ExitOK, code, stderr)

	stdout, stderr, code := runCLI(t, "--config", cfg, "records", "get", "--uri", "xdb://rc.t/samples/r1", "-o", "json")
	require.Equal(t, ExitOK, code, stderr)

	doc := jsonDoc(t, stdout)
	assert.Equal(t, "नमस्ते", doc["s"])
	assert.InDelta(t, -42, doc["i"], 0)
	assert.InDelta(t, 998877665544, doc["u"], 0)
	assert.InDelta(t, 0.07, doc["f"], 1e-12)
	assert.Equal(t, true, doc["b"])
	assert.Equal(t, "2024-06-01T10:00:00Z", doc["t"])
	assert.Equal(t, "aGVsbG8=", doc["by"])
	assert.Equal(t, []any{"silk", "handloom"}, doc["tags"])
	assert.Equal(t, []any{12999.5, 499.5}, doc["prices"])
}
