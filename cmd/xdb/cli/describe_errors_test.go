package cli

import (
	"encoding/json"
	"testing"

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

// TestDescribe_DaemonDown guards against describe.go's raw-error fallback:
// every describe variant that reaches the RPC client must render a
// CONNECTION_REFUSED envelope (not a bare "error: ..." line) and exit 2 when
// the daemon is unreachable, naming the resource/action that failed.
func TestDescribe_DaemonDown(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantResource string
		wantAction   string
	}{
		{"--methods", []string{"describe", "--methods"}, "introspect", "methods"},
		{"--types", []string{"describe", "--types"}, "introspect", "types"},
		{"--actions", []string{"describe", "--actions"}, "introspect", "actions"},
		{"method name", []string{"describe", "records.create"}, "introspect", "method"},
		{"type name", []string{"describe", "NoSuchType"}, "introspect", "type"},
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
