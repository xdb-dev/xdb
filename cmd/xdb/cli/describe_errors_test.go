package cli

import (
	"encoding/json"
	"testing"

	"github.com/xdb-dev/xdb/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// describeErrEnvelope mirrors [output.ErrorEnvelope]'s JSON shape for
// unmarshaling stderr in tests.
type describeErrEnvelope struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Hint     string `json:"hint"`
}

// TestDescribe_DaemonDown covers the one describe path with no offline
// fallback: data-schema lookups live in the daemon, so with it down the
// CLI must render a CONNECTION_REFUSED envelope and exit 2. (Method and
// type catalogs fall back to the embedded catalog instead — see
// TestDescribe_OfflineFallback.)
func TestDescribe_DaemonDown(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantResource string
		wantAction   string
	}{
		{"--uri", []string{"describe", "--uri", "xdb://a/b"}, "schemas", "describe"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath, _ := tempCLIConfig(t)
			args := append([]string{"--config", configPath}, tc.args...)

			stdout, stderr, code := runCLI(t, args...)

			assert.Equal(t, ExitConnection, code)
			assert.Empty(t, stdout)

			var env describeErrEnvelope
			require.NoError(t, json.Unmarshal([]byte(stderr), &env), "stderr: %s", stderr)

			assert.Equal(t, CodeConnectionRefused, env.Code)
			assert.Equal(t, tc.wantResource, env.Resource)
			assert.Equal(t, tc.wantAction, env.Action)
			assert.Contains(t, env.Hint, "daemon start")
		})
	}
}

// TestDescribe_BadURI_InvalidArgument verifies a malformed --uri fails fast
// with INVALID_ARGUMENT (exit 3) instead of reaching the RPC client.
func TestDescribe_BadURI_InvalidArgument(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", configPath, "describe", "--uri", "not-a-uri")

	assert.Equal(t, ExitInvalidArgs, code)
	assert.Empty(t, stdout)

	var env describeErrEnvelope
	require.NoError(t, json.Unmarshal([]byte(stderr), &env), "stderr: %s", stderr)
	assert.Equal(t, CodeInvalidArgument, env.Code)
	assert.Equal(t, "schemas", env.Resource)
	assert.Equal(t, "describe", env.Action)
}

func TestDescribe_OfflineFallback(t *testing.T) {
	cfg, _ := tempCLIConfig(t)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"methods", []string{"describe", "--methods"}, "records.create"},
		{"actions", []string{"describe", "--actions"}, "records"},
		{"types", []string{"describe", "--types"}, "Record"},
		{"method by name", []string{"describe", "records.create"}, "dry_run"},
		{"type by name", []string{"describe", "Record"}, "Record"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, code := runCLI(t, append([]string{"--config", cfg}, tt.args...)...)
			require.Equal(t, 0, code, "stderr: %s", stderr)
			assert.Contains(t, stdout, tt.want)
			assert.Contains(t, stdout, `"source"`, "offline results must be marked embedded")
			assert.Contains(t, stdout, "embedded")
		})
	}
}

func TestDescribe_LiveOmitsEmbeddedMarker(t *testing.T) {
	cfg := startCLITestDaemon(t)

	stdout, _, code := runCLI(t, "--config", cfg, "describe", "--methods")
	require.Equal(t, 0, code)
	assert.NotContains(t, stdout, "embedded")
}

func TestDescribe_UnknownMethodOffline(t *testing.T) {
	cfg, _ := tempCLIConfig(t)

	_, stderr, code := runCLI(t, "--config", cfg, "describe", "nosuch.action")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "NOT_FOUND")
}

func TestDescribe_CLIFlagsSection(t *testing.T) {
	cfg, _ := tempCLIConfig(t)

	stdout, _, code := runCLI(t, "--config", cfg, "describe", "records.delete")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, `"cli"`)
	assert.Contains(t, stdout, "force")
	assert.Contains(t, stdout, "quiet")

	stdout, _, code = runCLI(t, "--config", cfg, "describe", "records.list")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, "page-all")
	assert.Contains(t, stdout, "filter")
}

func TestDescribe_SchemaFormat(t *testing.T) {
	cfg, _ := tempCLIConfig(t)

	stdout, _, code := runCLI(t, "--config", cfg, "describe", "--schema-format")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, `"boolean"`)
	assert.NotContains(t, stdout, `"bool"`)
	assert.Contains(t, stdout, "strict")
	assert.Contains(t, stdout, "flexible")
	assert.Contains(t, stdout, "dynamic")
	assert.Contains(t, stdout, "elem_type")
	assert.Contains(t, stdout, "items")
}

func TestDescribe_NoArgsOverview(t *testing.T) {
	cfg, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", cfg, "describe")
	require.Equal(t, 0, code)
	assert.Empty(t, stderr)
	assert.Contains(t, stdout, "--schema-format")
	assert.Contains(t, stdout, "--actions")
	assert.Contains(t, stdout, "--filter")
}

func TestValueTypes_CompleteAndDescribed(t *testing.T) {
	require.Len(t, typeDescriptions, len(core.ValueTypes),
		"every core value type needs a description (and no stale extras)")

	for _, tid := range core.ValueTypes {
		assert.NotEmpty(t, typeDescriptions[tid], "type %s", tid)
	}
}
