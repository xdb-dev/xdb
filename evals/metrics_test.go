package evals

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bash(cmd string, failed bool, output string) Call {
	return Call{Tool: "Bash", Command: cmd, Failed: failed, Output: output}
}

const notFoundEnvelope = "Exit code 1\n" +
	`{"code":"NOT_FOUND","message":"x","resource":"records","action":"get",` +
	`"uri":"xdb://ns/s/1","hint":"try: xdb records list xdb://ns/s"}`

const forceEnvelope = "Exit code 3\n" +
	`{"code":"INVALID_ARGUMENT","message":"x","resource":"records","action":"delete",` +
	`"hint":"delete requires --force to confirm"}`

func TestComputeMetricsTotals(t *testing.T) {
	examples := map[string]bool{"records create": true}

	phase1 := &Trajectory{
		NumTurns: 5, DurationMS: 1500, CostUSD: 0.5, InputTokens: 10, OutputTokens: 20,
		Calls: []Call{
			{Tool: "Read", Output: "x"},
			bash("xdb --help", false, "..."),
			bash("xdb records create xdb://ns/s/1 --json '{}'", false, "{}"),
			bash("ls", true, "Exit code 1"),
		},
	}
	phase2 := &Trajectory{
		NumTurns: 2, DurationMS: 500, CostUSD: 0.25, InputTokens: 5, OutputTokens: 5,
		Calls: []Call{
			bash("xdb records create --help", false, "..."),
		},
	}

	m := ComputeMetrics(examples, phase1, phase2)

	assert.Equal(t, 7, m.Turns)
	assert.Equal(t, 5, m.ToolCalls)
	assert.Equal(t, 3, m.XDBCalls)
	assert.Equal(t, 2, m.DiscoveryCalls)
	assert.Equal(t, 0, m.FailedCalls)
	assert.InDelta(t, 2.0, m.DurationS, 1e-9)
	assert.InDelta(t, 0.75, m.CostUSD, 1e-9)
	assert.Equal(t, 15, m.InputTokens)
	assert.Equal(t, 25, m.OutputTokens)

	require.Len(t, m.Disclosure.Path, 2)
	assert.Equal(t, DiscoveryStep{Phase: 0, Command: "xdb --help", Layer: LayerOverview}, m.Disclosure.Path[0])
	assert.Equal(t, DiscoveryStep{Phase: 1, Command: "xdb records create --help", Layer: LayerAction}, m.Disclosure.Path[1])
	assert.Equal(t, []Layer{LayerOverview, LayerAction}, m.Disclosure.LayersVisited)
	assert.Equal(t, LayerAction, m.Disclosure.DeepestLayer)
	assert.Equal(t, 1, m.Disclosure.RedundantDiscovery)
}

func TestComputeMetricsEmpty(t *testing.T) {
	m := ComputeMetrics(nil, &Trajectory{})

	assert.Equal(t, LayerNone, m.Disclosure.DeepestLayer)
	assert.Equal(t, []Layer{}, m.Disclosure.LayersVisited)
	assert.Equal(t, []DiscoveryStep{}, m.Disclosure.Path)
	assert.InDelta(t, 0.0, m.RecoveryRate, 0)
}

func TestComputeMetricsRecoveries(t *testing.T) {
	tests := []struct {
		name              string
		calls             []Call
		want              Disclosure
		failed, recovered int
	}{
		{
			name: "blind failure then blind recovery",
			calls: []Call{
				bash("xdb records get xdb://ns/s/1", true, notFoundEnvelope),
				bash("xdb records get xdb://ns/s/2", false, "{}"),
			},
			want:      Disclosure{BlindFailures: 1, BlindRecoveries: 1, HintedFailures: 1},
			failed:    1,
			recovered: 1,
		},
		{
			name: "discovery recovery after action help",
			calls: []Call{
				bash("xdb records create xdb://ns/s/1 --data '{}'", true, "Exit code 3\nIncorrect Usage: flag provided but not defined: -data"),
				bash("xdb records create --help", false, "..."),
				bash("xdb records create xdb://ns/s/1 --json '{}'", false, "{}"),
			},
			want:      Disclosure{BlindFailures: 1, DiscoveryRecoveries: 1, UnknownFlagErrors: 1},
			failed:    1,
			recovered: 1,
		},
		{
			name: "covered by prior action help is not blind",
			calls: []Call{
				bash("xdb describe records.create", false, "..."),
				bash("xdb records create xdb://ns/s/1 --json '{'", true, "Exit code 1\nerror: bad json"),
			},
			want:   Disclosure{},
			failed: 1,
		},
		{
			name: "covered by resource help is not blind",
			calls: []Call{
				bash("xdb records --help", false, "..."),
				bash("xdb records update xdb://ns/s/1 --json '{'", true, "Exit code 1\nerror: bad json"),
			},
			want:   Disclosure{},
			failed: 1,
		},
		{
			name: "covered by root help is not blind",
			calls: []Call{
				bash("xdb --help", false, "..."),
				bash("xdb schemas create xdb://ns/s --json '{'", true, "Exit code 1\nerror: bad json"),
			},
			want:   Disclosure{},
			failed: 1,
		},
		{
			name: "skills cover everything",
			calls: []Call{
				bash("xdb skills get bulk-data", false, "..."),
				bash("xdb batch -f ops.ndjson", true, "Exit code 4\nerror"),
			},
			want:   Disclosure{},
			failed: 1,
		},
		{
			name: "recovery window is two non-discovery calls",
			calls: []Call{
				bash("xdb records get xdb://ns/s/1", true, "Exit code 1\nerror"),
				bash("xdb records list xdb://ns/s", false, "[]"),
				bash("xdb records list xdb://ns/s --limit 1", false, "[]"),
				bash("xdb records get xdb://ns/s/1", false, "{}"),
			},
			want:   Disclosure{BlindFailures: 1},
			failed: 1,
		},
		{
			name: "hint followed with command",
			calls: []Call{
				bash("xdb records get xdb://ns/s/1", true, notFoundEnvelope),
				bash("xdb records list xdb://ns/s -o ndjson", false, "[]"),
			},
			want:   Disclosure{BlindFailures: 1, HintedFailures: 1, HintFollowed: 1},
			failed: 1,
		},
		{
			name: "hint followed with flag",
			calls: []Call{
				bash("xdb records delete xdb://ns/s/1", true, forceEnvelope),
				bash("xdb records delete xdb://ns/s/1 --force", false, ""),
			},
			want:      Disclosure{BlindFailures: 1, HintedFailures: 1, HintFollowed: 1, BlindRecoveries: 1},
			failed:    1,
			recovered: 1,
		},
		{
			name: "hint ignored",
			calls: []Call{
				bash("xdb records delete xdb://ns/s/1", true, forceEnvelope),
				bash("xdb records get xdb://ns/s/1", false, "{}"),
			},
			want:   Disclosure{BlindFailures: 1, HintedFailures: 1},
			failed: 1,
		},
		{
			name: "non-xdb failures are not counted",
			calls: []Call{
				bash("jq . missing.json", true, "Exit code 2"),
			},
			want: Disclosure{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := ComputeMetrics(nil, &Trajectory{Calls: tt.calls})

			assert.Equal(t, tt.failed, m.FailedCalls, "failed")
			assert.Equal(t, tt.recovered, m.RecoveredCalls, "recovered")

			got := m.Disclosure
			got.Path, got.LayersVisited, got.DeepestLayer = nil, nil, 0
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExceededBudget(t *testing.T) {
	m := Metrics{FailedCalls: 5, CostUSD: 1.5, DiscoveryCalls: 2}
	m.Disclosure.BlindFailures = 2

	tests := []struct {
		name   string
		budget Budget
		want   []string
	}{
		{name: "none", budget: Budget{}, want: nil},
		{name: "under", budget: Budget{FailedCommands: 5, CostUSD: 2, BlindFailures: 2, DiscoveryCalls: 3}, want: nil},
		{
			name:   "over",
			budget: Budget{FailedCommands: 4, CostUSD: 1, BlindFailures: 1, DiscoveryCalls: 1},
			want: []string{
				"failed_commands 5 > 4",
				"blind_failures 2 > 1",
				"discovery_calls 2 > 1",
				"cost_usd 1.50 > 1.00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, m.ExceededBudget(tt.budget))
		})
	}
}
