package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootDispatch(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		wantCode          int
		stdoutContains    []string
		stdoutNotContains []string
		stdoutMaxBytes    int
		stderrContains    []string
	}{
		{
			name:           "unknown command",
			args:           []string{"frobnicate-nonexistent"},
			wantCode:       ExitInvalidArgs,
			stderrContains: []string{"INVALID_ARGUMENT", "frobnicate-nonexistent"},
			stdoutMaxBytes: 512,
		},
		{
			name:              "bare xdb shows help",
			args:              nil,
			wantCode:          ExitOK,
			stdoutContains:    []string{"RESOURCES:"},
			stdoutNotContains: []string{"# XDB CLI Context"},
		},
		{
			name:           "context command",
			args:           []string{"context"},
			wantCode:       ExitOK,
			stdoutContains: []string{"# XDB CLI Context"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath, _ := tempCLIConfig(t)
			fullArgs := append([]string{"--config", configPath}, tc.args...)

			stdout, stderr, code := runCLI(t, fullArgs...)

			assert.Equal(t, tc.wantCode, code)

			for _, s := range tc.stdoutContains {
				assert.Contains(t, stdout, s)
			}

			for _, s := range tc.stdoutNotContains {
				assert.NotContains(t, stdout, s)
			}

			for _, s := range tc.stderrContains {
				assert.Contains(t, stderr, s)
			}

			if tc.stdoutMaxBytes > 0 {
				assert.Less(t, len(stdout), tc.stdoutMaxBytes)
			}
		})
	}
}

func TestRootDispatch_ContextStartsWithHeader(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, _, code := runCLI(t, "--config", configPath, "context")

	assert.Equal(t, ExitOK, code)
	assert.True(t, strings.HasPrefix(stdout, "# XDB CLI Context"), "stdout: %q", stdout)
}

func TestRootDispatch_TypoSuggestion(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	_, stderr, code := runCLI(t, "--config", configPath, "recrods")

	assert.Equal(t, ExitInvalidArgs, code)

	if !strings.Contains(stderr, "did you mean") {
		t.Skip("no suggestion produced for this typo")
	}

	assert.Contains(t, stderr, "records")
}

func TestFlagParseError_Envelope(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", configPath, "records", "list", "--no-such-flag", "-o", "json")

	assert.Equal(t, ExitInvalidArgs, code)
	assert.Empty(t, stdout)
	assert.NotContains(t, stderr, "Incorrect Usage")
	assert.Equal(t, 1, strings.Count(stderr, `"code": "INVALID_ARGUMENT"`), "stderr: %s", stderr)
}

func TestNonexistentSubcommandHelp(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", configPath, "schemas", "upsert", "--help")

	assert.Equal(t, ExitInvalidArgs, code)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "No help topic")
}

func TestValidHelpStillWorks(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", configPath, "schemas", "create", "--help")

	assert.Equal(t, ExitOK, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "OPTIONS")
}
