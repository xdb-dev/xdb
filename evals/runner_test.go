package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClaude writes a claude stand-in that records its arguments and stdin
// and replays the smoke trajectory.
func fakeClaude(t *testing.T) (binary, argsFile, stdinFile string) {
	t.Helper()

	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args")
	stdinFile = filepath.Join(dir, "stdin")

	fixture, err := filepath.Abs("testdata/smoke.jsonl")
	require.NoError(t, err)

	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > " + argsFile + "\n" +
		"cat > " + stdinFile + "\n" +
		"cat " + fixture + "\n"

	binary = filepath.Join(dir, "claude")
	require.NoError(t, os.WriteFile(binary, []byte(script), 0o700))

	return binary, argsFile, stdinFile
}

func TestClaudeRunnerRunPhase(t *testing.T) {
	binary, argsFile, stdinFile := fakeClaude(t)
	runner := ClaudeRunner{Binary: binary}

	run, err := runner.RunPhase(t.Context(), PhaseRequest{
		Prompt:       "do the thing",
		SystemPrompt: "sys",
		Model:        "haiku",
		MaxTurns:     7,
		MaxBudgetUSD: 1.5,
		Dir:          t.TempDir(),
		Env:          os.Environ(),
	})
	require.NoError(t, err)

	assert.Equal(t, "ANSWER: done", run.Trajectory.FinalText)
	assert.NotEmpty(t, run.Raw)

	stdin, err := os.ReadFile(stdinFile)
	require.NoError(t, err)
	assert.Equal(t, "do the thing", string(stdin))

	args, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(args)), "\n")
	assert.Contains(t, lines, "--max-turns")
	assert.Contains(t, lines, "7")
	assert.Contains(t, lines, "--max-budget-usd")
	assert.Contains(t, lines, "1.50")
	assert.Contains(t, lines, "sys")
	assert.NotContains(t, lines, "--resume")
}

func TestClaudeRunnerArgsResume(t *testing.T) {
	args := ClaudeRunner{}.Args(PhaseRequest{Model: "sonnet", MaxTurns: 3, SessionID: "abc"})

	joined := strings.Join(args, " ")
	assert.Contains(t, joined, "--resume abc")
	assert.NotContains(t, joined, "--max-budget-usd")
	assert.Contains(t, joined, "--setting-sources  ")
}

func TestClaudeRunnerMissingBinary(t *testing.T) {
	_, err := ClaudeRunner{Binary: "/nonexistent/claude"}.RunPhase(t.Context(), PhaseRequest{Model: "x"})
	require.Error(t, err)
}
