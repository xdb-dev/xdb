package main

import (
	"testing"

	"github.com/sonnes/pi-go/pkg/agent"
	"github.com/sonnes/pi-go/pkg/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toolStart(id, name, command string) agent.Event {
	return agent.Event{
		Type:       agent.EventToolExecutionStart,
		ToolCallID: id,
		ToolName:   name,
		Args:       map[string]any{"command": command},
	}
}

func toolEnd(id, result string, isErr bool) agent.Event {
	return agent.Event{
		Type:       agent.EventToolExecutionEnd,
		ToolCallID: id,
		Result:     result,
		IsError:    isErr,
	}
}

func TestTraceCollectsCallsInOrder(t *testing.T) {
	tr := newTrace(10)

	for _, ev := range []agent.Event{
		{Type: agent.EventTurnStart},
		toolStart("a", "bash", "xdb schemas list"),
		toolEnd("a", "xdb://ns/things", false),
		{Type: agent.EventTurnStart},
		toolStart("b", "bash", "xdb records get xdb://ns/things/1"),
		toolEnd("b", "Exit code 1\n{\"code\":\"NOT_FOUND\"}", true),
		{Type: agent.EventAgentEnd, Usage: ai.Usage{Input: 100, Output: 20}},
	} {
		tr.add(ev)
	}

	assert.Equal(t, 2, tr.Turns)
	assert.False(t, tr.Truncated)
	assert.Equal(t, 100, tr.InputTokens)
	assert.Equal(t, 20, tr.OutputTokens)

	require.Len(t, tr.Calls, 2)
	assert.Equal(t, "bash", tr.Calls[0].Tool)
	assert.Equal(t, "xdb schemas list", tr.Calls[0].Command)
	assert.Equal(t, "xdb://ns/things", tr.Calls[0].Output)
	assert.False(t, tr.Calls[0].Failed)

	assert.True(t, tr.Calls[1].Failed)
	assert.Contains(t, tr.Calls[1].Output, "NOT_FOUND")
}

func TestTraceTracksTheTurnLimit(t *testing.T) {
	tr := newTrace(2)
	tr.add(agent.Event{Type: agent.EventTurnStart})
	assert.False(t, tr.Truncated)

	tr.add(agent.Event{Type: agent.EventTurnStart})
	assert.True(t, tr.Truncated, "a run that used every turn did not finish on its own")
}

func TestTraceCostSumsEveryCategory(t *testing.T) {
	tr := newTrace(0)
	tr.add(agent.Event{Type: agent.EventAgentEnd, Usage: ai.Usage{
		Cost: ai.UsageCost{Input: 0.01, Output: 0.02, CacheRead: 0.003, CacheWrite: 0.004},
	}})

	assert.InDelta(t, 0.037, tr.CostUSD, 1e-9)
}

func TestTraceRawIsOneJSONLinePerEvent(t *testing.T) {
	tr := newTrace(0)
	tr.add(toolStart("a", "bash", "xdb context"))
	tr.add(toolEnd("a", "ok", false))

	assert.Equal(t, 2, countLines(tr.Raw()))
}

func TestAnswerLine(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
		ok   bool
	}{
		{"plain", "done\nANSWER: 42", "42", true},
		{"last wins", "ANSWER: 1\nmore\nANSWER: 2", "2", true},
		{"strips markdown", "ANSWER: **42**", "42", true},
		{"absent", "no answer here", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := AnswerLine(tt.text)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func countLines(b []byte) int {
	n := 0

	for _, c := range b {
		if c == '\n' {
			n++
		}
	}

	return n
}
