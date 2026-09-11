package evals

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func passingResult(task string, turns int, cost float64) *TaskResult {
	r := &TaskResult{Task: task, Status: StatusPass}
	r.Metrics.Turns = turns
	r.Metrics.XDBCalls = 4
	r.Metrics.CostUSD = cost
	r.Metrics.Disclosure.DeepestLayer = LayerAction
	r.Phases = []PhaseResult{{Name: "p", Status: StatusPass, Checks: []CheckResult{{Name: "c", Status: StatusPass}}}}

	return r
}

func TestSummarizeAndMarkdown(t *testing.T) {
	failing := &TaskResult{Task: "ledger", Status: StatusFail, Reason: `phase "use": check "x": boom`}
	failing.Metrics.Turns = 30
	failing.Metrics.FailedCalls = 2
	failing.Metrics.RecoveryRate = 0.5
	failing.Metrics.Disclosure.DeepestLayer = LayerNone
	failing.Metrics.Disclosure.BlindFailures = 1
	failing.Phases = []PhaseResult{{Name: "use", Status: StatusFail, Checks: []CheckResult{
		{Name: "x", Status: StatusFail, Error: "boom"},
		{Name: "y", Status: StatusPass},
	}}}

	results := []*TaskResult{
		passingResult("ledger", 10, 0.5),
		passingResult("ledger", 20, 0.7),
		failing,
		passingResult("store", 8, 0.2),
	}

	summaries := Summarize(results)
	require.Len(t, summaries, 2)

	ledger := summaries[0]
	assert.Equal(t, "ledger", ledger.Task)
	assert.Equal(t, 3, ledger.Runs)
	assert.Equal(t, 2, ledger.Passed)
	assert.Equal(t, 3, ledger.ChecksPassed)
	assert.Equal(t, 4, ledger.ChecksTotal)
	assert.Equal(t, 20, ledger.Turns, "median of 10, 20, 30")
	assert.Equal(t, LayerAction, ledger.DeepestLayer)
	assert.Equal(t, []string{`phase "use": check "x": boom`}, ledger.Reasons)

	md := Markdown(summaries)
	assert.Contains(t, md, "| ledger | 2/3 PASS | 3/4 | 20 |")
	assert.Contains(t, md, "| store | PASS | 1/1 | 8 | 4 | 0 | 0 | L2 | 0% | $0.20 | - |")
	assert.Contains(t, md, "2 tasks: 1 PASS, 1 FAIL")
	assert.Contains(t, md, "- ledger: phase \"use\"")
}

func TestWriteSummary(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, WriteSummary(dir, []*TaskResult{passingResult("a", 1, 0)}))

	assert.FileExists(t, filepath.Join(dir, "summary.md"))
	assert.FileExists(t, filepath.Join(dir, "summary.json"))
}
