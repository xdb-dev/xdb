package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTable(t *testing.T) {
	results := []*TaskResult{
		{
			Task:   "ledger",
			Status: StatusPass,
			Phases: []PhaseResult{{Checks: []CheckResult{{Status: StatusPass}, {Status: StatusPass}}}},
			Metrics: Metrics{
				Turns: 17, XDBCalls: 10, FailedCalls: 1, RecoveryRate: 1, CostUSD: 0.21,
				Disclosure: Disclosure{BlindFailures: 1, DeepestLayer: LayerNone},
			},
		},
		{
			Task:   "citations",
			Status: StatusFail,
			Reason: `phase "cite": check "ten citations" failed`,
			Phases: []PhaseResult{{Checks: []CheckResult{{Status: StatusPass}, {Status: StatusFail}}}},
			Metrics: Metrics{
				Turns: 16, XDBCalls: 12, FailedCalls: 2, CostUSD: 0.08,
				Disclosure: Disclosure{BlindFailures: 2, DeepestLayer: LayerReference},
			},
		},
	}

	out := Table(results)

	assert.Contains(t, out, "ledger")
	assert.Contains(t, out, "2/2")
	assert.Contains(t, out, "$0.21")
	assert.Contains(t, out, "-", "a task that asked for no layer shows a dash")
	assert.Contains(t, out, "L3")
	assert.Contains(t, out, "2 tasks: 1 PASS, 1 FAIL")
	assert.Contains(t, out, `- citations: phase "cite"`)
}

func TestWriteSummary(t *testing.T) {
	dir := t.TempDir()
	results := []*TaskResult{{Task: "ledger", Status: StatusPass}}

	require.NoError(t, WriteSummary(dir, results))

	for _, name := range []string{"summary.json", "summary.txt"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		assert.Contains(t, string(data), "ledger")
	}
}
