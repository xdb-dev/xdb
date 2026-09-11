package evals

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptedRunner replays one trajectory per phase and records the requests.
// It runs the Bash commands of the trajectory in the sandbox, so the state
// checks see their effect.
type scriptedRunner struct {
	phases   []*Trajectory
	requests []PhaseRequest
	sandbox  *Sandbox
}

func (r *scriptedRunner) RunPhase(ctx context.Context, req PhaseRequest) (*PhaseRun, error) {
	r.requests = append(r.requests, req)
	traj := r.phases[len(r.requests)-1]

	for _, call := range traj.Calls {
		if call.Tool == "Bash" && r.sandbox != nil {
			_, _ = r.sandbox.Run(ctx, call.Command)
		}
	}

	return &PhaseRun{Trajectory: traj, Raw: []byte("{}\n")}, nil
}

const runTaskYAML = `
name: two-phase
namespace: eval
model: haiku
max_turns: 5
budget:
  failed_commands: 2
phases:
  - name: setup
    prompt: Make a schema in $NS.
    checks:
      - name: schema exists
        run: xdb schemas get xdb://$NS/things -o json
        expect: { exit: 0 }
  - name: use
    prompt: Add a thing and answer.
    checks:
      - name: answer
        answer: "1"
      - name: one record
        run: xdb records list xdb://$NS/things -o ndjson
        expect: { ndjson_count: 1, ndjson_ids: [t1] }
`

func TestRunTaskPass(t *testing.T) {
	binary := testBinary(t)
	dir := writeTask(t, t.TempDir(), "two-phase", runTaskYAML)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "fixtures"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fixtures", "in.csv"), []byte("x\n"), 0o600))

	task, err := LoadTask(dir)
	require.NoError(t, err)

	runner := &scriptedRunner{}
	runner.phases = []*Trajectory{
		{
			SessionID: "sess-1", Subtype: "success", NumTurns: 2, CostUSD: 0.1,
			Calls: []Call{
				bash("xdb --help", false, ""),
				bash(`xdb schemas create xdb://eval/things --json '{"fields":{"n":{"type":"integer"}}}'`, false, ""),
			},
		},
		{
			SessionID: "sess-1", Subtype: "success", NumTurns: 3, CostUSD: 0.2, FinalText: "Done.\nANSWER: 1",
			Calls: []Call{
				bash(`xdb records create xdb://eval/things/t1 --json '{"n":1}'`, false, ""),
			},
		},
	}

	results := t.TempDir()
	opts := Options{Binary: binary, Runner: runner, ResultsDir: results}

	// The runner needs the sandbox to run commands. RunTask creates the
	// sandbox, so the hook reads it from the request directory on the first
	// call.
	runner.sandbox = nil
	result, err := runTaskWithSandboxHook(t, opts, task, runner)
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
	assert.Empty(t, result.StrayProcesses)

	// Phase 2 resumed the session of phase 1. Both phases got the task
	// prompt with $NS expanded, and the guide in the system prompt.
	require.Len(t, runner.requests, 2)
	assert.Empty(t, runner.requests[0].SessionID)
	assert.Equal(t, "sess-1", runner.requests[1].SessionID)
	assert.Equal(t, "Make a schema in eval.", runner.requests[0].Prompt)
	assert.Equal(t, "haiku", runner.requests[0].Model)
	assert.Equal(t, 5, runner.requests[0].MaxTurns)
	assert.Contains(t, runner.requests[0].SystemPrompt, "in.csv")
	assert.Contains(t, runner.requests[0].SystemPrompt, "# XDB CLI Context")
	assert.NotContains(t, runner.requests[0].SystemPrompt, "$NS")

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

	runner := &scriptedRunner{phases: []*Trajectory{
		{SessionID: "s", Subtype: "success", Calls: []Call{bash("echo nothing", false, "")}},
		{SessionID: "s", Subtype: "success"},
	}}

	result, err := runTaskWithSandboxHook(t, Options{Binary: binary, Runner: runner}, task, runner)
	require.NoError(t, err)

	assert.Equal(t, StatusFail, result.Status)
	assert.Contains(t, result.Reason, `phase "setup"`)
	assert.Contains(t, result.Reason, `check "schema exists"`)
	require.Len(t, result.Phases, 2)
	assert.Equal(t, StatusFail, result.Phases[0].Status)
	assert.Equal(t, StatusNotRun, result.Phases[1].Status)
	assert.Len(t, runner.requests, 1, "phase 2 never ran")
}

func TestRunTaskMaxTurnsFailsPhase(t *testing.T) {
	binary := testBinary(t)
	dir := writeTask(t, t.TempDir(), "two-phase", runTaskYAML)

	task, err := LoadTask(dir)
	require.NoError(t, err)

	runner := &scriptedRunner{phases: []*Trajectory{
		{SessionID: "s", Subtype: "error_max_turns"},
		{SessionID: "s", Subtype: "success"},
	}}

	result, err := runTaskWithSandboxHook(t, Options{Binary: binary, Runner: runner}, task, runner)
	require.NoError(t, err)

	assert.Equal(t, StatusFail, result.Status)
	assert.Contains(t, result.Reason, "error_max_turns")
	assert.Equal(t, StatusNotRun, result.Phases[0].Checks[0].Status)
}

func TestRunTaskBudgetFails(t *testing.T) {
	binary := testBinary(t)
	dir := writeTask(t, t.TempDir(), "two-phase", runTaskYAML)

	task, err := LoadTask(dir)
	require.NoError(t, err)

	failing := bash("xdb records get xdb://eval/things/none", true, "Exit code 1\n{}")
	runner := &scriptedRunner{phases: []*Trajectory{
		{
			SessionID: "s", Subtype: "success",
			Calls: []Call{
				failing, failing, failing,
				bash(`xdb schemas create xdb://eval/things --json '{"fields":{"n":{"type":"integer"}}}'`, false, ""),
			},
		},
		{
			SessionID: "s", Subtype: "success", FinalText: "ANSWER: 1",
			Calls: []Call{bash(`xdb records create xdb://eval/things/t1 --json '{"n":1}'`, false, "")},
		},
	}}

	result, err := runTaskWithSandboxHook(t, Options{Binary: binary, Runner: runner}, task, runner)
	require.NoError(t, err)

	assert.Equal(t, StatusFail, result.Status)
	assert.Equal(t, []string{"failed_commands 3 > 2"}, result.BudgetExceeded)
	assert.Contains(t, result.Reason, "budget")
}

// runTaskWithSandboxHook runs the task and gives the scripted runner the
// sandbox. Then the runner can run the commands of the trajectory for real.
func runTaskWithSandboxHook(t *testing.T, opts Options, task *Task, runner *scriptedRunner) (*TaskResult, error) {
	t.Helper()

	hooked := &hookRunner{inner: runner}
	opts.Runner = hooked

	return RunTask(t.Context(), opts, task)
}

type hookRunner struct {
	inner *scriptedRunner
}

func (h *hookRunner) RunPhase(ctx context.Context, req PhaseRequest) (*PhaseRun, error) {
	if h.inner.sandbox == nil {
		root := filepath.Dir(req.Dir)
		h.inner.sandbox = &Sandbox{
			Root: root,
			Home: filepath.Join(root, "home"),
			Bin:  filepath.Join(root, "bin"),
			Work: req.Dir,
		}
	}

	return h.inner.RunPhase(ctx, req)
}
