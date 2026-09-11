package evals

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTrajectorySmoke(t *testing.T) {
	f, err := os.Open("testdata/smoke.jsonl")
	require.NoError(t, err)
	defer f.Close()

	traj, err := ParseTrajectory(f)
	require.NoError(t, err)

	assert.Equal(t, "1de691b1-a93e-473e-8f3f-4d6cf22c1aca", traj.SessionID)
	assert.Equal(t, "success", traj.Subtype)
	assert.Equal(t, 3, traj.NumTurns)
	assert.Equal(t, int64(4482), traj.DurationMS)
	assert.InDelta(t, 0.0182873, traj.CostUSD, 1e-9)
	assert.Equal(t, 18, traj.InputTokens)
	assert.Equal(t, 279, traj.OutputTokens)
	assert.Equal(t, "ANSWER: done", traj.FinalText)

	// Calls are in issue order. The parser links results by id, also when
	// they arrive out of order.
	require.Len(t, traj.Calls, 2)
	assert.Equal(t, "Bash", traj.Calls[0].Tool)
	assert.Equal(t, "ls /nonexistent-dir-xyz", traj.Calls[0].Command)
	assert.True(t, traj.Calls[0].Failed)
	assert.Contains(t, traj.Calls[0].Output, "No such file")
	assert.Equal(t, "echo ok", traj.Calls[1].Command)
	assert.False(t, traj.Calls[1].Failed)
	assert.Equal(t, "ok", traj.Calls[1].Output)
}

func TestParseTrajectoryMaxTurns(t *testing.T) {
	in := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"s1"}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/x"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":[{"type":"text","text":"line"}],"is_error":false}]}}`,
		`{"type":"result","subtype":"error_max_turns","session_id":"s1","num_turns":4,"duration_ms":10,"total_cost_usd":0.1,"usage":{"input_tokens":1,"output_tokens":2}}`,
	}, "\n")

	traj, err := ParseTrajectory(strings.NewReader(in))
	require.NoError(t, err)

	assert.Equal(t, "error_max_turns", traj.Subtype)
	assert.Empty(t, traj.FinalText)
	require.Len(t, traj.Calls, 1)
	assert.Equal(t, "Read", traj.Calls[0].Tool)
	assert.Empty(t, traj.Calls[0].Command)
	assert.Equal(t, "line", traj.Calls[0].Output)
}

func TestParseTrajectoryNoResult(t *testing.T) {
	in := `{"type":"system","subtype":"init","session_id":"s1"}` + "\n"

	_, err := ParseTrajectory(strings.NewReader(in))
	require.ErrorIs(t, err, ErrNoResult)
}

func TestParseTrajectoryBadJSON(t *testing.T) {
	_, err := ParseTrajectory(strings.NewReader("{not json\n"))
	require.Error(t, err)
}

func TestAnswerLine(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
		ok   bool
	}{
		{name: "last line", text: "Loaded 12 rows.\n\nANSWER: 4880000", want: "4880000", ok: true},
		{name: "trims", text: "ANSWER:   done  \n", want: "done", ok: true},
		{name: "last wins", text: "ANSWER: 1\nANSWER: 2", want: "2", ok: true},
		{name: "code fence", text: "```\nANSWER: 3\n```", want: "3", ok: true},
		{name: "bold", text: "**ANSWER: 3**", want: "3", ok: true},
		{name: "missing", text: "no answer here", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := AnswerLine(tt.text)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
