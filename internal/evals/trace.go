package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sonnes/pi-go/pkg/agent"
	"github.com/sonnes/pi-go/pkg/ai"
)

// Trace is the record of one phase: what the subject did, what it cost, and
// what it said last. [ComputeMetrics] reads it.
type Trace struct {
	// FinalText is the last reply of the subject. Answer checks read it.
	FinalText string
	Turns     int
	// Truncated is true when the subject used every turn it was given. Such
	// a run stopped because of the limit, not because it was done.
	Truncated    bool
	DurationS    float64
	CostUSD      float64
	InputTokens  int
	OutputTokens int
	// Calls lists every tool call in the order the subject made it.
	Calls []Call

	maxTurns int
	byID     map[string]int
	raw      bytes.Buffer
	started  time.Time
}

// Call is one tool call and its result.
type Call struct {
	Tool string
	// Command is the shell command of a bash call, empty otherwise.
	Command string
	// Output is the text the tool returned, or the error text when the tool
	// failed. An xdb error envelope arrives here.
	Output string
	// Failed is true when the tool reported an error. For bash, that is a
	// non-zero exit code.
	Failed bool
}

// newTrace starts a trace for a run limited to maxTurns. A maxTurns of 0 is
// no limit.
func newTrace(maxTurns int) *Trace {
	return &Trace{
		maxTurns: maxTurns,
		byID:     map[string]int{},
		started:  time.Now(),
	}
}

// add folds one agent event into the trace and appends it to the raw log.
func (t *Trace) add(ev agent.Event) {
	if line, err := json.Marshal(ev); err == nil {
		t.raw.Write(line)
		t.raw.WriteByte('\n')
	}

	switch ev.Type {
	case agent.EventTurnStart:
		t.Turns++

		if t.maxTurns > 0 && t.Turns >= t.maxTurns {
			t.Truncated = true
		}

	case agent.EventToolExecutionStart:
		t.byID[ev.ToolCallID] = len(t.Calls)
		t.Calls = append(t.Calls, Call{
			Tool:    ev.ToolName,
			Command: commandArg(ev.Args),
		})

	case agent.EventToolExecutionEnd:
		i, ok := t.byID[ev.ToolCallID]
		if !ok {
			return
		}

		t.Calls[i].Output = fmt.Sprint(ev.Result)
		t.Calls[i].Failed = ev.IsError

	case agent.EventAgentEnd:
		t.InputTokens += ev.Usage.Input + ev.Usage.CacheRead + ev.Usage.CacheWrite
		t.OutputTokens += ev.Usage.Output + ev.Usage.Reasoning
		t.CostUSD += totalCost(ev.Usage)
		t.FinalText = lastAssistantText(ev.Messages)
		t.DurationS = time.Since(t.started).Seconds()
	}
}

// Raw returns the event log as NDJSON, one event per line.
func (t *Trace) Raw() []byte { return t.raw.Bytes() }

// commandArg reads the shell command out of a bash tool call.
func commandArg(args map[string]any) string {
	cmd, _ := args["command"].(string)

	return cmd
}

// totalCost sums every cost category. [ai.UsageCost] holds no total.
func totalCost(u ai.Usage) float64 {
	c := u.Cost

	return c.Input + c.Output + c.CacheRead + c.CacheWrite +
		c.Reasoning + c.InputAudio + c.OutputAudio
}

// lastAssistantText returns the text of the last assistant message.
func lastAssistantText(msgs []ai.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != ai.RoleAssistant {
			continue
		}

		if text := msgs[i].JoinText("\n"); text != "" {
			return text
		}
	}

	return ""
}

var answerRe = regexp.MustCompile(`(?m)^\s*ANSWER:\s*(.+?)\s*$`)

// AnswerLine returns the value of the last ANSWER line in text. The subject
// writes one when the task asks a question. Markdown emphasis around the
// value is stripped, because a model adds it unasked.
func AnswerLine(text string) (string, bool) {
	matches := answerRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return "", false
	}

	last := matches[len(matches)-1][1]

	return strings.Trim(last, " \t*`_"), true
}
