package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const judgeEnvelope = `{"type":"result","subtype":"success","total_cost_usd":0.05,` +
	`"result":"{\"scores\":[{\"criterion\":\"money\",\"score\":1,\"note\":\"strings\"}]}",` +
	`"structured_output":{"scores":[{"criterion":"money","score":2,"note":"strings"},{"criterion":"dates","score":4,"note":"time"}]}}`

func TestParseJudgeOutput(t *testing.T) {
	res, err := parseJudgeOutput([]byte(judgeEnvelope), "")
	require.NoError(t, err)

	require.Len(t, res.Scores, 2)
	assert.Equal(t, 2, res.Scores[0].Score)
	assert.InDelta(t, 3.0, res.Average, 1e-9)
	assert.InDelta(t, 0.05, res.CostUSD, 1e-9)
}

func TestParseJudgeOutputFallsBackToResult(t *testing.T) {
	in := `{"subtype":"success","result":"{\"scores\":[{\"criterion\":\"x\",\"score\":5,\"note\":\"n\"}]}"}`

	res, err := parseJudgeOutput([]byte(in), "")
	require.NoError(t, err)
	assert.InDelta(t, 5.0, res.Average, 1e-9)
}

func TestParseJudgeOutputErrors(t *testing.T) {
	_, err := parseJudgeOutput([]byte("nope"), "boom")
	require.ErrorContains(t, err, "boom")

	_, err = parseJudgeOutput([]byte(`{"subtype":"error_max_turns","result":""}`), "")
	require.ErrorContains(t, err, "no scores")
}

func TestClaudeJudgeScore(t *testing.T) {
	dir := t.TempDir()
	stdinFile := filepath.Join(dir, "stdin")
	script := "#!/bin/sh\ncat > " + stdinFile + "\nprintf '%s' '" + judgeEnvelope + "'\n"
	binary := filepath.Join(dir, "claude")
	require.NoError(t, os.WriteFile(binary, []byte(script), 0o700))

	judge := ClaudeJudge{Binary: binary}
	res, err := judge.Score(t.Context(), JudgeRequest{Rubric: "- money", Evidence: "amount: string", Dir: dir, Env: os.Environ()})
	require.NoError(t, err)
	assert.InDelta(t, 3.0, res.Average, 1e-9)

	prompt, err := os.ReadFile(stdinFile)
	require.NoError(t, err)
	assert.Contains(t, string(prompt), "# Rubric\n\n- money")
	assert.Contains(t, string(prompt), "amount: string")

	args := strings.Join(judge.Args(JudgeRequest{}), " ")
	assert.Contains(t, args, "--model sonnet")
	assert.Contains(t, args, "--restricted")
	assert.Contains(t, args, "--json-schema")
}

func TestBuildEvidence(t *testing.T) {
	binary := testBinary(t)

	sb, err := NewSandbox(binary, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sb.Close() })

	ctx := t.Context()
	require.NoError(t, sb.Init(ctx))

	out, err := sb.Run(ctx, `xdb schemas create xdb://ev/things --json '{"fields":{"n":{"type":"integer"}}}' && xdb records create xdb://ev/things/a --json '{"n":1}'`)
	require.NoError(t, err)
	require.Equal(t, 0, out.Exit, out.Stderr)

	task := &Task{Namespace: "ev"}
	traj := &Trajectory{Calls: []Call{
		bash("xdb schemas create xdb://ev/things --json '{}'", false, ""),
		bash("xdb records get xdb://ev/things/zzz", true, ""),
		bash("jq . x", false, ""),
	}}

	evidence := BuildEvidence(ctx, sb, task, []*Trajectory{traj})

	assert.Contains(t, evidence, "### xdb://ev/things")
	assert.Contains(t, evidence, `"n"`)
	assert.Contains(t, evidence, `"_id":"a"`)
	assert.Contains(t, evidence, "[ok] xdb schemas create")
	assert.Contains(t, evidence, "[FAILED] xdb records get")
	assert.NotContains(t, evidence, "jq . x")
}
