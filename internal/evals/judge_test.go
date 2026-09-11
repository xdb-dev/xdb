package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJudgePromptCarriesRubricAndEvidence(t *testing.T) {
	p := OpenRouterJudge{}.Prompt(JudgeRequest{Rubric: "- money", Evidence: "amount: string"})

	assert.Contains(t, p.System, "1 (not at all) to 5 (fully)")
	require.Len(t, p.Messages, 1)

	text := p.Messages[0].JoinText("\n")
	assert.Contains(t, text, "# Rubric\n\n- money")
	assert.Contains(t, text, "amount: string")
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
	traj := &Trace{Calls: []Call{
		bash("xdb schemas create xdb://ev/things --json '{}'", false, ""),
		bash("xdb records get xdb://ev/things/zzz", true, ""),
		bash("jq . x", false, ""),
	}}

	evidence := BuildEvidence(ctx, sb, task, []*Trace{traj})

	assert.Contains(t, evidence, "### xdb://ev/things")
	assert.Contains(t, evidence, `"n"`)
	assert.Contains(t, evidence, `"_id":"a"`)
	assert.Contains(t, evidence, "[ok] xdb schemas create")
	assert.Contains(t, evidence, "[FAILED] xdb records get")
	assert.NotContains(t, evidence, "jq . x")
}
