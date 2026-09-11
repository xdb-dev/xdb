package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

// Table renders one row per result, then a one-line total and the reason of
// each failure.
//
// There is no aggregation across runs. To compare runs, run the command more
// than once and read the result files.
func Table(results []*TaskResult) string {
	var b strings.Builder

	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintln(w, "TASK\tRESULT\tCHECKS\tTURNS\tXDB\tFAILED\tBLIND\tDEEPEST\tRECOVERY\tCOST")

	passed := 0

	for _, r := range results {
		if r.Status == StatusPass {
			passed++
		}

		m := r.Metrics
		checked, total := r.ChecksPassed()

		_, _ = fmt.Fprintf(w, "%s\t%s\t%d/%d\t%d\t%d\t%d\t%d\t%s\t%.0f%%\t$%.2f\n",
			r.Task,
			r.Status,
			checked, total,
			m.Turns,
			m.XDBCalls,
			m.FailedCalls,
			m.Disclosure.BlindFailures,
			layerCell(m.Disclosure.DeepestLayer),
			m.RecoveryRate*100,
			m.CostUSD,
		)
	}

	_ = w.Flush()

	fmt.Fprintf(&b, "\n%d tasks: %d PASS, %d FAIL\n", len(results), passed, len(results)-passed)

	for _, r := range results {
		if r.Reason != "" {
			fmt.Fprintf(&b, "- %s: %s\n", r.Task, r.Reason)
		}
	}

	return b.String()
}

// WriteSummary writes summary.txt and summary.json into dir.
func WriteSummary(dir string, results []*TaskResult) error {
	if err := writeJSON(filepath.Join(dir, "summary.json"), results); err != nil {
		return err
	}

	path := filepath.Join(dir, "summary.txt")
	if err := os.WriteFile(path, []byte(Table(results)), 0o600); err != nil {
		return fmt.Errorf("[xdb/evals] write summary.txt: %w", err)
	}

	return nil
}

// layerCell renders a disclosure layer, or a dash when the agent asked for
// no layer at all.
func layerCell(l Layer) string {
	if l == LayerNone {
		return "-"
	}

	return fmt.Sprintf("L%d", l)
}
