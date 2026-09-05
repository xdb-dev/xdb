package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/cmd/xdb/daemon"
)

func seedPageRecords(t *testing.T, cfg string, n int) {
	t.Helper()

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://page.t/items",
		"--json", `{"fields":{"name":{"type":"string"}}}`)
	require.Equal(t, 0, code)

	for i := range n {
		_, _, code := runCLI(t, "--config", cfg, "records", "create",
			"--uri", fmt.Sprintf("xdb://page.t/items/i%d", i),
			"--json", fmt.Sprintf(`{"name":"n%d"}`, i))
		require.Equal(t, 0, code)
	}
}

func TestRecordsList_PageEnvelope(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedPageRecords(t, cfg, 5)

	stdout, _, code := runCLI(t, "--config", cfg, "records", "list",
		"--uri", "xdb://page.t/items", "--limit", "2", "-o", "json")
	require.Equal(t, 0, code)

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	assert.Equal(t, float64(5), doc["total"])
	assert.Equal(t, float64(2), doc["next_offset"])
	assert.Len(t, doc["items"], 2)
}

func TestRecordsList_PageAll(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedPageRecords(t, cfg, 5)

	stdout, _, code := runCLI(t, "--config", cfg, "records", "list",
		"--uri", "xdb://page.t/items", "--limit", "2", "--page-all", "-o", "ndjson")
	require.Equal(t, 0, code)

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	assert.Len(t, lines, 5)

	stdout, _, code = runCLI(t, "--config", cfg, "records", "list",
		"--uri", "xdb://page.t/items", "--limit", "2", "--page-all", "-o", "json")
	require.Equal(t, 0, code)

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	assert.Len(t, doc["items"], 5)
	assert.NotContains(t, doc, "next_offset")
}

func TestRecordsList_NdjsonStaysBare(t *testing.T) {
	cfg := startCLITestDaemon(t)
	seedPageRecords(t, cfg, 2)

	stdout, _, code := runCLI(t, "--config", cfg, "records", "list",
		"--uri", "xdb://page.t/items", "-o", "ndjson")
	require.Equal(t, 0, code)
	assert.NotContains(t, stdout, `"total"`)

	for line := range strings.SplitSeq(strings.TrimSpace(stdout), "\n") {
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m))
		assert.Contains(t, m, "_id")
	}
}

func TestDaemonStatus_ExitCodes(t *testing.T) {
	t.Run("running exits 0", func(t *testing.T) {
		// Status is PID-file based; simulate a live daemon with this
		// test process's own PID.
		cfg, dir := tempCLIConfig(t)
		// The PID file is named after the socket, so derive it the same
		// way the daemon does rather than hard-coding the default name.
		pidPath := daemon.PIDPath(filepath.Join(dir, "test.sock"))
		require.NoError(t, daemon.WritePID(pidPath))

		_, _, code := runCLI(t, "--config", cfg, "daemon", "status")
		assert.Equal(t, 0, code)
	})

	t.Run("stopped exits 2 with status output", func(t *testing.T) {
		stopped, _ := tempCLIConfig(t)
		stdout, stderr, code := runCLI(t, "--config", stopped, "daemon", "status")
		assert.Equal(t, 2, code)
		assert.Contains(t, stdout, `"stopped"`)
		assert.Empty(t, stderr)
	})

	t.Run("stopped quiet exits 2 silently", func(t *testing.T) {
		stopped, _ := tempCLIConfig(t)
		stdout, stderr, code := runCLI(t, "--config", stopped, "daemon", "status", "--quiet")
		assert.Equal(t, 2, code)
		assert.Empty(t, stdout)
		assert.Empty(t, stderr)
	})
}

func TestAliasDepthValidation(t *testing.T) {
	cfg := startCLITestDaemon(t)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"rm namespace URI", []string{"rm", "xdb://onlyns", "--force"}, "rm needs"},
		{"put schema URI", []string{"put", "xdb://ns/schema", "--json", `{}`}, "put needs"},
		{"make-schema record URI", []string{"make-schema", "xdb://ns/schema/id", "--json", `{}`}, "make-schema needs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, code := runCLI(t, append([]string{"--config", cfg}, tt.args...)...)
			assert.Equal(t, 3, code)
			assert.Contains(t, stderr, tt.want)
		})
	}
}

func TestBigIntegersRenderVerbatim(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://big.t/nums",
		"--json", `{"fields":{"big":{"type":"integer"}}}`)
	require.Equal(t, 0, code)

	stdout, _, code := runCLI(t, "--config", cfg, "records", "create",
		"--uri", "xdb://big.t/nums/n1",
		"--json", `{"big":9007199254740993}`, "-o", "json")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, "9007199254740993", "create response must not round through float64")

	stdout, _, code = runCLI(t, "--config", cfg, "records", "list",
		"--uri", "xdb://big.t/nums", "-o", "ndjson")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, "9007199254740993", "list items must not round through float64")

	stdout, _, code = runCLI(t, "--config", cfg, "export", "--uri", "xdb://big.t/nums")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, "9007199254740993", "export must not round through float64")
}

func TestNamespaceGet_RendersSchemaTree(t *testing.T) {
	cfg := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", cfg, "schemas", "create",
		"--uri", "xdb://nsget.t/things",
		"--json", `{"fields":{"name":{"type":"string"}}}`)
	require.Equal(t, 0, code)

	stdout, _, code := runCLI(t, "--config", cfg, "get", "xdb://nsget.t", "-o", "json")
	require.Equal(t, 0, code)

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	assert.Equal(t, "nsget.t", doc["namespace"])
	assert.Equal(t, float64(1), doc["total_schemas"])
	assert.Contains(t, doc["schemas"], "xdb://nsget.t/things")
}
