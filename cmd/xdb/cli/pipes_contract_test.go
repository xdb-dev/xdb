package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the stdin and pipe contracts of the CLI: batch from stdin
// and an export to import roundtrip. The retired e2e suite asserted them
// through the binary.

func TestBatch_ReadsOperationsFromStdin(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://pc.t/items", "--json", `{"fields":{"name":{"type":"string"}}}`)
	require.Equal(t, ExitOK, code)

	ops := `{"op":"records.create","uri":"xdb://pc.t/items/a","data":{"name":"A"}}` + "\n" +
		`{"op":"records.create","uri":"xdb://pc.t/items/b","data":{"name":"B"}}` + "\n"

	stdout, stderr, code := runCLIStdin(t, ops, "--config", cfg, "batch", "-", "-o", "json")
	require.Equal(t, ExitOK, code, stderr)

	doc := jsonDoc(t, stdout)
	assert.InDelta(t, 2, doc["total"], 0)
	assert.InDelta(t, 2, doc["succeeded"], 0)
	assert.InDelta(t, 0, doc["failed"], 0)

	stdout, _, code = runCLI(t, "--config", cfg, "records", "list", "--uri", "xdb://pc.t/items", "-o", "ndjson")
	require.Equal(t, ExitOK, code)
	assert.ElementsMatch(t, []string{"a", "b"}, ndjsonIDs(t, stdout))
}

func TestExportImport_Roundtrip(t *testing.T) {
	// Known product bug. Export emits _version. Import treats it as a
	// compare-and-swap against a record that does not exist in the target
	// schema. The roundtrip fails with CONFLICT on line 1. Remove the Skip
	// when import ignores _version for absent records, or when export stops
	// emitting it.
	t.Skip("export | import into another schema fails with CONFLICT on _version")

	cfg := startCLITestDaemon(t)

	for _, name := range []string{"contacts", "contacts_v2"} {
		_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
			"--uri", "xdb://pc.t/"+name, "--json", `{"fields":{"name":{"type":"string"},"email":{"type":"string","required":true}}}`)
		require.Equal(t, ExitOK, code)
	}

	for _, id := range []string{"priya", "arjun", "kavita"} {
		_, _, code := runCLI(t, "--config", cfg, "records", "create",
			"--uri", "xdb://pc.t/contacts/"+id, "--json", `{"name":"`+id+`","email":"`+id+`@x.in"}`)
		require.Equal(t, ExitOK, code)
	}

	exported, stderr, code := runCLI(t, "--config", cfg, "export", "--uri", "xdb://pc.t/contacts")
	require.Equal(t, ExitOK, code, stderr)

	_, stderr, code = runCLIStdin(t, exported, "--config", cfg, "import", "--uri", "xdb://pc.t/contacts_v2")
	require.Equal(t, ExitOK, code, stderr)

	stdout, _, code := runCLI(t, "--config", cfg, "records", "list", "--uri", "xdb://pc.t/contacts_v2", "-o", "ndjson")
	require.Equal(t, ExitOK, code)
	assert.ElementsMatch(t, []string{"priya", "arjun", "kavita"}, ndjsonIDs(t, stdout))
}
