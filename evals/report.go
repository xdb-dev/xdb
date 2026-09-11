package evals

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TaskSummary aggregates the runs of one task. With one run, the medians
// are the values of that run.
type TaskSummary struct {
	Task          string  `json:"task"`
	Runs          int     `json:"runs"`
	Passed        int     `json:"passed"`
	ChecksPassed  int     `json:"checks_passed"`
	ChecksTotal   int     `json:"checks_total"`
	Turns         int     `json:"turns"`
	XDBCalls      int     `json:"xdb_calls"`
	FailedCalls   int     `json:"failed_calls"`
	BlindFailures int     `json:"blind_failures"`
	DeepestLayer  Layer   `json:"deepest_layer"`
	RecoveryRate  float64 `json:"recovery_rate"`
	CostUSD       float64 `json:"cost_usd"`
	// RubricAverage is the median rubric average, or 0 when no judge ran.
	RubricAverage  float64  `json:"rubric_average"`
	Reasons        []string `json:"reasons,omitempty"`
	StrayProcesses []string `json:"stray_processes,omitempty"`
}

// Summarize groups results by task, sorted by task name.
func Summarize(results []*TaskResult) []TaskSummary {
	byTask := map[string][]*TaskResult{}
	for _, r := range results {
		byTask[r.Task] = append(byTask[r.Task], r)
	}

	names := make([]string, 0, len(byTask))
	for name := range byTask {
		names = append(names, name)
	}

	sort.Strings(names)

	summaries := make([]TaskSummary, 0, len(names))
	for _, name := range names {
		summaries = append(summaries, summarizeTask(name, byTask[name]))
	}

	return summaries
}

func summarizeTask(name string, runs []*TaskResult) TaskSummary {
	s := TaskSummary{Task: name, Runs: len(runs), DeepestLayer: LayerNone}

	n := len(runs)
	turns := make([]int, 0, n)
	xdbCalls := make([]int, 0, n)
	failed := make([]int, 0, n)
	blind := make([]int, 0, n)
	recovery := make([]float64, 0, n)
	cost := make([]float64, 0, n)
	rubric := make([]float64, 0, n)

	for _, r := range runs {
		if r.Status == StatusPass {
			s.Passed++
		} else if r.Reason != "" {
			s.Reasons = append(s.Reasons, r.Reason)
		}

		passed, total := r.ChecksPassed()
		s.ChecksPassed += passed
		s.ChecksTotal += total

		m := r.Metrics
		turns = append(turns, m.Turns)
		xdbCalls = append(xdbCalls, m.XDBCalls)
		failed = append(failed, m.FailedCalls)
		blind = append(blind, m.Disclosure.BlindFailures)
		recovery = append(recovery, m.RecoveryRate)
		cost = append(cost, m.CostUSD)

		if r.Rubric != nil {
			rubric = append(rubric, r.Rubric.Average)
		}

		if m.Disclosure.DeepestLayer > s.DeepestLayer {
			s.DeepestLayer = m.Disclosure.DeepestLayer
		}

		s.StrayProcesses = append(s.StrayProcesses, r.StrayProcesses...)
	}

	s.Turns = medianInt(turns)
	s.XDBCalls = medianInt(xdbCalls)
	s.FailedCalls = medianInt(failed)
	s.BlindFailures = medianInt(blind)
	s.RecoveryRate = medianFloat(recovery)
	s.CostUSD = medianFloat(cost)
	s.RubricAverage = medianFloat(rubric)

	return s
}

// Markdown returns the summary table and the one-line total as Markdown
// text.
func Markdown(summaries []TaskSummary) string {
	var b strings.Builder

	b.WriteString("| Task | Result | Checks | Turns | xdb calls | Failed | Blind | Deepest | Recovery | Cost | Rubric |\n")
	b.WriteString("|------|--------|--------|------:|----------:|-------:|------:|--------:|---------:|-----:|-------:|\n")

	tasksPassed := 0

	for _, s := range summaries {
		if s.Passed == s.Runs {
			tasksPassed++
		}

		fmt.Fprintf(&b, "| %s | %s | %d/%d | %d | %d | %d | %d | %s | %.0f%% | $%.2f | %s |\n",
			s.Task,
			resultCell(s),
			s.ChecksPassed, s.ChecksTotal,
			s.Turns,
			s.XDBCalls,
			s.FailedCalls,
			s.BlindFailures,
			layerCell(s.DeepestLayer),
			s.RecoveryRate*100,
			s.CostUSD,
			rubricCell(s.RubricAverage),
		)
	}

	fmt.Fprintf(&b, "\n%d tasks: %d PASS, %d FAIL\n", len(summaries), tasksPassed, len(summaries)-tasksPassed)

	for _, s := range summaries {
		for _, reason := range s.Reasons {
			fmt.Fprintf(&b, "- %s: %s\n", s.Task, reason)
		}

		for _, proc := range s.StrayProcesses {
			fmt.Fprintf(&b, "- %s: stray process: %s\n", s.Task, proc)
		}
	}

	return b.String()
}

// WriteSummary writes summary.md and summary.json into dir.
func WriteSummary(dir string, results []*TaskResult) error {
	summaries := Summarize(results)

	if err := writeJSON(filepath.Join(dir, "summary.json"), summaries); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, "summary.md"), []byte(Markdown(summaries)), 0o600); err != nil {
		return fmt.Errorf("[xdb/evals] write summary.md: %w", err)
	}

	return nil
}

func resultCell(s TaskSummary) string {
	if s.Runs == 1 {
		if s.Passed == 1 {
			return StatusPass
		}

		return StatusFail
	}

	return fmt.Sprintf("%d/%d PASS", s.Passed, s.Runs)
}

func rubricCell(avg float64) string {
	if avg == 0 {
		return "-"
	}

	return fmt.Sprintf("%.1f/5", avg)
}

func layerCell(l Layer) string {
	if l == LayerNone {
		return "-"
	}

	return fmt.Sprintf("L%d", l)
}

func medianInt(xs []int) int {
	if len(xs) == 0 {
		return 0
	}

	sorted := append([]int(nil), xs...)
	sort.Ints(sorted)

	return sorted[len(sorted)/2]
}

func medianFloat(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}

	sorted := append([]float64(nil), xs...)
	sort.Float64s(sorted)

	return sorted[len(sorted)/2]
}
