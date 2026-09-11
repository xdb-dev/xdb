package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptedSubject replays one trace per phase. It runs the shell commands
// of each trace in the sandbox, so the state checks see their effect.
type scriptedSubject struct {
	phases  []*Trace
	sb      *Sandbox
	spec    SubjectSpec
	prompts []string
	opens   int
	ran     int
}

func (s *scriptedSubject) Run(ctx context.Context, prompt string) (*Trace, error) {
	s.prompts = append(s.prompts, prompt)

	trace := s.phases[s.ran]
	s.ran++

	for _, call := range trace.Calls {
		if call.Tool == shellToolName {
			_, _ = s.sb.Run(ctx, call.Command)
		}
	}

	return trace, nil
}

func (s *scriptedSubject) Close() error { return nil }

// open returns the [OpenSubject] that hands the subject its sandbox.
func (s *scriptedSubject) open() OpenSubject {
	return func(sb *Sandbox, spec SubjectSpec) (Subject, error) {
		s.sb = sb
		s.spec = spec
		s.opens++

		return s, nil
	}
}

const runTaskYAML = `
name: two-phase
namespace: eval
model: anthropic/claude-haiku-4.5
max_turns: 5
budget:
  failed_commands: 2
phases:
  - name: setup
    prompt: Make a schema in $NS.
    checks:
      - name: schema exists
        run: xdb schemas get xdb://$NS/things -o json
  - name: use
    prompt: Add a thing and answer.
    checks:
      - name: answer
        answer: "1"
      - name: one record
        run: test "$(xdb records list xdb://$NS/things -o ndjson | wc -l)" -eq 1
`

func TestRunTaskPass(t *testing.T) {
	binary := testBinary(t)
	dir := writeTask(t, t.TempDir(), "two-phase", runTaskYAML)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "fixtures"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fixtures", "in.csv"), []byte("x\n"), 0o600))

	task, err := LoadTask(dir)
	require.NoError(t, err)

	subject := &scriptedSubject{}
	subject.phases = []*Trace{
		{
			Turns: 2, CostUSD: 0.1,
			Calls: []Call{
				bash("xdb --help", false, ""),
				bash(`xdb schemas create xdb://eval/things --json '{"fields":{"n":{"type":"integer"}}}'`, false, ""),
			},
		},
		{
			Turns: 3, CostUSD: 0.2, FinalText: "Done.\nANSWER: 1",
			Calls: []Call{
				bash(`xdb records create xdb://eval/things/t1 --json '{"n":1}'`, false, ""),
			},
		},
	}

	results := t.TempDir()
	opts := Options{Binary: binary, ResultsDir: results}

	opts.Open = subject.open()
	result, err := RunTask(t.Context(), opts, task)
	require.NoError(t, err)

	assert.Equal(t, StatusPass, result.Status, result.Reason)
	require.Len(t, result.Phases, 2)
	assert.Equal(t, StatusPass, result.Phases[0].Status)
	assert.Equal(t, StatusPass, result.Phases[1].Status)

	passed, total := result.ChecksPassed()
	assert.Equal(t, 3, passed)
	assert.Equal(t, 3, total)

	assert.Equal(t, 5, result.Metrics.Turns)
	assert.Equal(t, 3, result.Metrics.XDBCalls)
	assert.Equal(t, 1, result.Metrics.DiscoveryCalls)
	assert.InDelta(t, 0.3, result.Metrics.CostUSD, 1e-9)

	// One subject ran both phases, so phase 2 saw the conversation of
	// phase 1. Each prompt is the task prompt with $NS expanded.
	assert.Equal(t, 1, subject.opens)
	require.Len(t, subject.prompts, 2)
	assert.Equal(t, "Make a schema in eval.", subject.prompts[0])
	assert.Equal(t, "Add a thing and answer.", subject.prompts[1])

	// The spec carries the task settings and the guide.
	assert.Equal(t, "anthropic/claude-haiku-4.5", subject.spec.Model)
	assert.Equal(t, 5, subject.spec.MaxTurns)
	assert.Contains(t, subject.spec.SystemPrompt, "in.csv")
	assert.Contains(t, subject.spec.SystemPrompt, "# XDB CLI Context")
	assert.NotContains(t, subject.spec.SystemPrompt, "$NS")

	// Results are on disk.
	assert.FileExists(t, filepath.Join(results, "trajectory.01-setup.jsonl"))
	assert.FileExists(t, filepath.Join(results, "trajectory.02-use.jsonl"))
	assert.FileExists(t, filepath.Join(results, "result.json"))
}

func TestRunTaskFailStopsLaterPhases(t *testing.T) {
	binary := testBinary(t)
	dir := writeTask(t, t.TempDir(), "two-phase", runTaskYAML)

	task, err := LoadTask(dir)
	require.NoError(t, err)

	subject := &scriptedSubject{phases: []*Trace{
		{Calls: []Call{bash("echo nothing", false, "")}},
		{},
	}}

	result, err := RunTask(t.Context(), Options{Binary: binary, Open: subject.open()}, task)
	require.NoError(t, err)

	assert.Equal(t, StatusFail, result.Status)
	assert.Contains(t, result.Reason, `phase "setup"`)
	assert.Contains(t, result.Reason, `check "schema exists"`)
	require.Len(t, result.Phases, 2)
	assert.Equal(t, StatusFail, result.Phases[0].Status)
	assert.Equal(t, StatusNotRun, result.Phases[1].Status)
	assert.Equal(t, 1, subject.ran, "phase 2 never ran")
}

func TestRunTaskMaxTurnsFailsPhase(t *testing.T) {
	binary := testBinary(t)
	dir := writeTask(t, t.TempDir(), "two-phase", runTaskYAML)

	task, err := LoadTask(dir)
	require.NoError(t, err)

	subject := &scriptedSubject{phases: []*Trace{
		{Truncated: true},
		{},
	}}

	result, err := RunTask(t.Context(), Options{Binary: binary, Open: subject.open()}, task)
	require.NoError(t, err)

	assert.Equal(t, StatusFail, result.Status)
	assert.Contains(t, result.Reason, "every one of its")
	assert.Equal(t, StatusNotRun, result.Phases[0].Checks[0].Status)
}

func TestRunTaskBudgetFails(t *testing.T) {
	binary := testBinary(t)
	dir := writeTask(t, t.TempDir(), "two-phase", runTaskYAML)

	task, err := LoadTask(dir)
	require.NoError(t, err)

	failing := bash("xdb records get xdb://eval/things/none", true, "Exit code 1\n{}")
	subject := &scriptedSubject{phases: []*Trace{
		{
			Calls: []Call{
				failing, failing, failing,
				bash(`xdb schemas create xdb://eval/things --json '{"fields":{"n":{"type":"integer"}}}'`, false, ""),
			},
		},
		{
			FinalText: "ANSWER: 1",
			Calls:     []Call{bash(`xdb records create xdb://eval/things/t1 --json '{"n":1}'`, false, "")},
		},
	}}

	result, err := RunTask(t.Context(), Options{Binary: binary, Open: subject.open()}, task)
	require.NoError(t, err)

	assert.Equal(t, StatusFail, result.Status)
	assert.Equal(t, []string{"failed_commands 3 > 2"}, result.BudgetExceeded)
	assert.Contains(t, result.Reason, "budget")
}
