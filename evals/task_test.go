package evals

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validTaskYAML = `
name: household-ledger
description: Model and run a household ledger.
tags: [ledger, money]
namespace: household
model: haiku
max_turns: 30
budget:
  failed_commands: 8
  blind_failures: 3
  cost_usd: 2.5
phases:
  - name: setup
    prompt: |
      Track my household finances.
    checks:
      - name: accounts schema exists
        run: xdb schemas get xdb://household/accounts -o json
        expect: { exit: 0 }
  - name: month-end
    prompt: Load the statements.
    checks:
      - name: hdfc balance
        answer: "4880000"
      - name: any balance
        answer_regex: '^\d+$'
      - name: twelve transactions
        run: xdb records list xdb://household/transactions -o ndjson
        expect:
          exit: 0
          ndjson_count: 12
          ndjson_ids: [a, b]
          json: { _id: x, nested: { k: 1 } }
          error: { code: NOT_FOUND }
rubric: |
  Money is stored as integers.
`

func writeTask(t *testing.T, root, name, body string) string {
	t.Helper()

	dir := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "task.yaml"), []byte(body), 0o600))

	return dir
}

func TestLoadTask(t *testing.T) {
	dir := writeTask(t, t.TempDir(), "household-ledger", validTaskYAML)

	task, err := LoadTask(dir)
	require.NoError(t, err)

	assert.Equal(t, "household-ledger", task.Name)
	assert.Equal(t, dir, task.Dir)
	assert.Equal(t, "household", task.Namespace)
	assert.Equal(t, "haiku", task.Model)
	assert.Equal(t, 30, task.MaxTurns)
	assert.Equal(t, 8, task.Budget.FailedCommands)
	assert.Equal(t, 3, task.Budget.BlindFailures)
	assert.InDelta(t, 2.5, task.Budget.CostUSD, 0)
	assert.Equal(t, "Money is stored as integers.\n", task.Rubric)

	require.Len(t, task.Phases, 2)
	assert.Equal(t, "setup", task.Phases[0].Name)
	assert.Equal(t, "Track my household finances.\n", task.Phases[0].Prompt)
	require.Len(t, task.Phases[0].Checks, 1)
	assert.Equal(t, "xdb schemas get xdb://household/accounts -o json", task.Phases[0].Checks[0].Run)
	require.NotNil(t, task.Phases[0].Checks[0].Expect.Exit)
	assert.Equal(t, 0, *task.Phases[0].Checks[0].Expect.Exit)

	checks := task.Phases[1].Checks
	require.Len(t, checks, 3)
	require.NotNil(t, checks[0].Answer)
	assert.Equal(t, "4880000", *checks[0].Answer)
	assert.Equal(t, `^\d+$`, checks[1].AnswerRegex)
	require.NotNil(t, checks[2].Expect.NDJSONCount)
	assert.Equal(t, 12, *checks[2].Expect.NDJSONCount)
	assert.Equal(t, []string{"a", "b"}, checks[2].Expect.NDJSONIDs)
	assert.Equal(t, map[string]any{"_id": "x", "nested": map[string]any{"k": 1}}, checks[2].Expect.JSON)
	assert.Equal(t, map[string]any{"code": "NOT_FOUND"}, checks[2].Expect.Error)
}

func TestLoadTaskDefaults(t *testing.T) {
	dir := writeTask(t, t.TempDir(), "minimal", `
name: minimal
namespace: ns
phases:
  - name: only
    prompt: do it
    checks:
      - name: a
        answer: "1"
`)

	task, err := LoadTask(dir)
	require.NoError(t, err)

	assert.Equal(t, DefaultModel, task.Model)
	assert.Equal(t, DefaultMaxTurns, task.MaxTurns)
}

func TestLoadTaskInvalid(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "missing name",
			yaml: "namespace: ns\nphases: [{name: p, prompt: x, checks: [{name: c, answer: '1'}]}]\n",
			want: "name is required",
		},
		{
			name: "bad name",
			yaml: "name: Bad_Name\nnamespace: ns\nphases: [{name: p, prompt: x, checks: [{name: c, answer: '1'}]}]\n",
			want: "name must match",
		},
		{
			name: "missing namespace",
			yaml: "name: t\nphases: [{name: p, prompt: x, checks: [{name: c, answer: '1'}]}]\n",
			want: "namespace is required",
		},
		{
			name: "no phases",
			yaml: "name: t\nnamespace: ns\n",
			want: "at least one phase",
		},
		{
			name: "phase without prompt",
			yaml: "name: t\nnamespace: ns\nphases: [{name: p, checks: [{name: c, answer: '1'}]}]\n",
			want: `phase "p": prompt is required`,
		},
		{
			name: "phase without name",
			yaml: "name: t\nnamespace: ns\nphases: [{prompt: x}]\n",
			want: "phase 1: name is required",
		},
		{
			name: "duplicate phase",
			yaml: "name: t\nnamespace: ns\nphases: [{name: p, prompt: x}, {name: p, prompt: y}]\n",
			want: `phase "p": duplicate name`,
		},
		{
			name: "check without name",
			yaml: "name: t\nnamespace: ns\nphases: [{name: p, prompt: x, checks: [{answer: '1'}]}]\n",
			want: `phase "p" check 1: name is required`,
		},
		{
			name: "check with run and answer",
			yaml: "name: t\nnamespace: ns\nphases: [{name: p, prompt: x, checks: [{name: c, run: xdb, expect: {exit: 0}, answer: '1'}]}]\n",
			want: `check "c": exactly one of run, answer, answer_regex`,
		},
		{
			name: "check with nothing",
			yaml: "name: t\nnamespace: ns\nphases: [{name: p, prompt: x, checks: [{name: c}]}]\n",
			want: `check "c": exactly one of run, answer, answer_regex`,
		},
		{
			name: "run check without expect",
			yaml: "name: t\nnamespace: ns\nphases: [{name: p, prompt: x, checks: [{name: c, run: xdb}]}]\n",
			want: `check "c": expect needs at least one assertion`,
		},
		{
			name: "bad answer regex",
			yaml: "name: t\nnamespace: ns\nphases: [{name: p, prompt: x, checks: [{name: c, answer_regex: '('}]}]\n",
			want: `check "c": answer_regex`,
		},
		{
			name: "unknown key",
			yaml: "name: t\nnamespace: ns\nphases: [{name: p, prompt: x, checks: [{name: c, run: xdb, expect: {ndjson_cnt: 1}}]}]\n",
			want: "ndjson_cnt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeTask(t, t.TempDir(), "t", tt.yaml)

			_, err := LoadTask(dir)
			require.ErrorIs(t, err, ErrInvalidTask)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestLoadTaskMissingFile(t *testing.T) {
	_, err := LoadTask(t.TempDir())
	require.Error(t, err)
}

func TestLoadTasks(t *testing.T) {
	root := t.TempDir()
	writeTask(t, root, "zeta", "name: zeta\nnamespace: ns\nphases: [{name: p, prompt: x}]\n")
	writeTask(t, root, "alpha", "name: alpha\nnamespace: ns\nphases: [{name: p, prompt: x}]\n")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "not-a-task"), 0o750))

	tasks, err := LoadTasks(root)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	assert.Equal(t, "alpha", tasks[0].Name)
	assert.Equal(t, "zeta", tasks[1].Name)
}

func TestLoadTasksNameMustMatchDir(t *testing.T) {
	root := t.TempDir()
	writeTask(t, root, "dir-name", "name: other\nnamespace: ns\nphases: [{name: p, prompt: x}]\n")

	_, err := LoadTasks(root)
	require.ErrorIs(t, err, ErrInvalidTask)
	assert.ErrorContains(t, err, "directory")
}
