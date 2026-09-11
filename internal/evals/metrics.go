package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// recoveryWindow is the number of non-discovery xdb calls after a failure
// that can count as its recovery.
const recoveryWindow = 2

// Metrics is the friction summary of one task, summed over its phases.
// FailedCalls and RecoveredCalls count xdb calls only. A failed `jq` is not
// a CLI fault.
type Metrics struct {
	Turns          int     `json:"turns"`
	ToolCalls      int     `json:"tool_calls"`
	XDBCalls       int     `json:"xdb_calls"`
	DiscoveryCalls int     `json:"discovery_calls"`
	FailedCalls    int     `json:"failed_calls"`
	RecoveredCalls int     `json:"recovered_calls"`
	RecoveryRate   float64 `json:"recovery_rate"`
	DurationS      float64 `json:"duration_s"`
	CostUSD        float64 `json:"cost_usd"`
	InputTokens    int     `json:"input_tokens"`
	OutputTokens   int     `json:"output_tokens"`

	Disclosure Disclosure `json:"disclosure"`
}

// Disclosure measures how deep the agent had to dig, and whether the layers
// and the error hints helped. The raw event log holds every command, so this
// holds only what a budget reads or a finding cited.
type Disclosure struct {
	// DeepestLayer is the deepest layer the agent reached.
	DeepestLayer Layer `json:"deepest_layer"`
	// BlindFailures counts failed xdb calls on an action that the agent did
	// not look up before the call. These calls count as a lookup:
	//   - root help, describe --actions, or any skills call
	//   - help on the same resource
	//   - help on the same action
	BlindFailures int `json:"blind_failures"`
	// HintedFailures counts failures whose error envelope had a hint.
	HintedFailures int `json:"hinted_failures"`
	// HintFollowed counts hinted failures where the next xdb call did what
	// the hint said.
	HintFollowed int `json:"hint_followed"`
}

// ExceededBudget lists every budget that the metrics exceed. A zero budget
// value is no limit.
func (m Metrics) ExceededBudget(b Budget) []string {
	var over []string

	if b.FailedCommands > 0 && m.FailedCalls > b.FailedCommands {
		over = append(over, fmt.Sprintf("failed_commands %d > %d", m.FailedCalls, b.FailedCommands))
	}

	if b.BlindFailures > 0 && m.Disclosure.BlindFailures > b.BlindFailures {
		over = append(over, fmt.Sprintf("blind_failures %d > %d", m.Disclosure.BlindFailures, b.BlindFailures))
	}

	if b.DiscoveryCalls > 0 && m.DiscoveryCalls > b.DiscoveryCalls {
		over = append(over, fmt.Sprintf("discovery_calls %d > %d", m.DiscoveryCalls, b.DiscoveryCalls))
	}

	if b.CostUSD > 0 && m.CostUSD > b.CostUSD {
		over = append(over, fmt.Sprintf("cost_usd %.2f > %.2f", m.CostUSD, b.CostUSD))
	}

	return over
}

// xdbEvent is one classified xdb call with its position in the trajectory.
type xdbEvent struct {
	call Call
	xdb  XDBCall
}

// ComputeMetrics sums the metrics of the given phase traces.
func ComputeMetrics(phases ...*Trace) Metrics {
	m := Metrics{}
	m.Disclosure.DeepestLayer = LayerNone

	for _, traj := range phases {
		m.Turns += traj.Turns
		m.ToolCalls += len(traj.Calls)
		m.DurationS += traj.DurationS
		m.CostUSD += traj.CostUSD
		m.InputTokens += traj.InputTokens
		m.OutputTokens += traj.OutputTokens

		events := classifyCalls(traj.Calls)
		m.XDBCalls += len(events)

		walker := &phaseWalker{m: &m}
		walker.walk(events)
	}

	if m.FailedCalls > 0 {
		m.RecoveryRate = float64(m.RecoveredCalls) / float64(m.FailedCalls)
	}

	return m
}

func classifyCalls(calls []Call) []xdbEvent {
	events := make([]xdbEvent, 0, len(calls))

	for _, call := range calls {
		if call.Tool != shellToolName {
			continue
		}

		xdb, ok := ClassifyCommand(call.Command)
		if !ok {
			continue
		}

		events = append(events, xdbEvent{call: call, xdb: xdb})
	}

	return events
}

// phaseWalker walks the xdb events of one phase. It accumulates the lookup
// coverage and the disclosure counters.
type phaseWalker struct {
	m *Metrics

	general   bool
	resources map[string]bool
	actions   map[string]bool
}

func (w *phaseWalker) walk(events []xdbEvent) {
	w.resources = map[string]bool{}
	w.actions = map[string]bool{}

	for i, ev := range events {
		if ev.xdb.Discovery {
			w.discover(ev)

			continue
		}

		if !ev.call.Failed {
			continue
		}

		w.fail(ev, events[i+1:])
	}
}

func (w *phaseWalker) discover(ev xdbEvent) {
	d := &w.m.Disclosure
	w.m.DiscoveryCalls++

	if ev.xdb.Layer > d.DeepestLayer {
		d.DeepestLayer = ev.xdb.Layer
	}

	switch ev.xdb.Layer {
	case LayerContext, LayerSkills:
		w.general = true
	case LayerOverview:
		if ev.xdb.Resource == "" || ev.xdb.Resource == "describe" {
			w.general = true
		} else {
			w.resources[ev.xdb.Resource] = true
		}
	case LayerAction:
		w.actions[ev.xdb.Key()] = true
	case LayerReference:
		if ev.xdb.Action != "" {
			w.actions[ev.xdb.Key()] = true
		}
	case LayerNone:
	}
}

func (w *phaseWalker) fail(ev xdbEvent, rest []xdbEvent) {
	d := &w.m.Disclosure
	w.m.FailedCalls++

	covered := w.general || w.resources[ev.xdb.Resource] || w.actions[ev.xdb.Key()]
	if !covered {
		d.BlindFailures++
	}

	if hint := envelopeHint(ev.call.Output); hint != "" {
		d.HintedFailures++

		if next, ok := nextXDB(rest); ok && hintFollowed(hint, ev.xdb, next) {
			d.HintFollowed++
		}
	}

	if findRecovery(ev.xdb, rest) {
		w.m.RecoveredCalls++
	}
}

func nextXDB(rest []xdbEvent) (xdbEvent, bool) {
	if len(rest) == 0 {
		return xdbEvent{}, false
	}

	return rest[0], true
}

// findRecovery looks for a successful call on the same action inside the
// recovery window.
func findRecovery(failed XDBCall, rest []xdbEvent) bool {
	seen := 0

	for _, ev := range rest {
		if ev.xdb.Discovery {
			continue
		}

		seen++

		if ev.xdb.Key() == failed.Key() && !ev.call.Failed {
			return true
		}

		if seen == recoveryWindow {
			break
		}
	}

	return false
}

// envelopeHint extracts the hint field from an error envelope in a tool
// output. An "Exit code N" line can come before the JSON.
func envelopeHint(output string) string {
	start := strings.Index(output, "{")
	if start < 0 {
		return ""
	}

	var env struct {
		Hint string `json:"hint"`
	}

	dec := json.NewDecoder(strings.NewReader(output[start:]))
	if err := dec.Decode(&env); err != nil {
		return ""
	}

	return env.Hint
}

var (
	hintCommandRe = regexp.MustCompile(`xdb .*?(?: to | for |$)`)
	hintFlagRe    = regexp.MustCompile(`--[a-z-]+`)
	hintActionRe  = regexp.MustCompile(`\b(get|list|create|update|upsert|delete)\b`)
)

// hintFollowed reports whether the next xdb call obeyed the hint. That is
// the suggested command, the suggested flag on the same action, or a
// suggested action.
func hintFollowed(hint string, failed XDBCall, next xdbEvent) bool {
	if m := hintCommandRe.FindString(hint); m != "" {
		suggested := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(m, " to"), " for"))

		return strings.Contains(next.call.Command, suggested)
	}

	if flag := hintFlagRe.FindString(hint); flag != "" {
		return next.xdb.Key() == failed.Key() && strings.Contains(next.call.Command, flag)
	}

	for _, m := range hintActionRe.FindAllString(hint, -1) {
		if next.xdb.Action == m && next.xdb.Resource == failed.Resource {
			return true
		}
	}

	return false
}
