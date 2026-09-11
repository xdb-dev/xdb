package evals

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Status values for tasks, phases, and checks.
const (
	StatusPass   = "PASS"
	StatusFail   = "FAIL"
	StatusNotRun = "NOT_RUN"
)

// Options configures one task run.
type Options struct {
	// Binary is the xdb binary to copy into the sandbox.
	Binary string
	Runner Runner
	// ResultsDir receives the trajectories, checks, and metrics of this
	// run. Empty means write nothing.
	ResultsDir string
	// If Model is set, it overrides the task model.
	Model string
	// If Judge is set, it scores the run against the task rubric. Nil means
	// no rubric score.
	Judge Judge
}

// TaskResult is the graded outcome of one task run.
type TaskResult struct {
	Task    string        `json:"task"`
	Status  string        `json:"status"`
	Reason  string        `json:"reason,omitempty"`
	Phases  []PhaseResult `json:"phases"`
	Metrics Metrics       `json:"metrics"`
	// Rubric is the verdict of the judge, when a judge ran and the task has
	// a rubric.
	Rubric *RubricResult `json:"rubric,omitempty"`
	// BudgetExceeded lists the budgets that the run exceeded.
	BudgetExceeded []string `json:"budget_exceeded,omitempty"`
	// StrayProcesses lists sandbox processes that are still alive after
	// teardown.
	StrayProcesses []string `json:"stray_processes,omitempty"`
}

// ChecksPassed returns passed and total check counts across phases.
func (r *TaskResult) ChecksPassed() (passed, total int) {
	for _, p := range r.Phases {
		for _, c := range p.Checks {
			total++

			if c.Status == StatusPass {
				passed++
			}
		}
	}

	return passed, total
}

// PhaseResult is the outcome of one phase.
type PhaseResult struct {
	Name    string        `json:"name"`
	Status  string        `json:"status"`
	Reason  string        `json:"reason,omitempty"`
	Subtype string        `json:"subtype,omitempty"`
	Checks  []CheckResult `json:"checks"`
}

// CheckResult is the outcome of one check.
type CheckResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// RunTask runs every phase of the task in a fresh sandbox and grades it. An
// error means that the harness could not run the task (no sandbox, no
// daemon, or no subject). If the subject fails the task, the result is FAIL
// and the error is nil.
func RunTask(ctx context.Context, opts Options, task *Task) (*TaskResult, error) {
	sb, err := NewSandbox(opts.Binary, filepath.Join(task.Dir, "fixtures"))
	if err != nil {
		return nil, err
	}

	result, runErr := runInSandbox(ctx, opts, task, sb)

	closeErr := sb.Close()

	if runErr != nil {
		return nil, runErr
	}

	if closeErr != nil {
		return nil, closeErr
	}

	stray, err := sb.StrayProcesses(ctx)
	if err != nil {
		return nil, err
	}

	result.StrayProcesses = stray

	if opts.ResultsDir != "" {
		if err := writeJSON(filepath.Join(opts.ResultsDir, "result.json"), result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func runInSandbox(ctx context.Context, opts Options, task *Task, sb *Sandbox) (*TaskResult, error) {
	if err := sb.Init(ctx); err != nil {
		return nil, err
	}

	guide, err := sb.Context(ctx)
	if err != nil {
		return nil, err
	}

	model := task.Model
	if opts.Model != "" {
		model = opts.Model
	}

	result := &TaskResult{Task: task.Name, Status: StatusPass}
	trajectories := make([]*Trajectory, 0, len(task.Phases))
	sessionID := ""

	for i, phase := range task.Phases {
		if result.Status == StatusFail {
			result.Phases = append(result.Phases, PhaseResult{Name: phase.Name, Status: StatusNotRun})

			continue
		}

		req := PhaseRequest{
			Prompt:       expand(phase.Prompt, task),
			SystemPrompt: BuildSystemPrompt(sb, guide),
			Model:        model,
			MaxTurns:     task.MaxTurns,
			MaxBudgetUSD: task.Budget.CostUSD,
			SessionID:    sessionID,
			Dir:          sb.Work,
			Env:          sb.Env(),
		}

		run, err := opts.Runner.RunPhase(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("phase %q: %w", phase.Name, err)
		}

		if err := writeRaw(opts.ResultsDir, i, phase.Name, run.Raw); err != nil {
			return nil, err
		}

		sessionID = run.Trajectory.SessionID
		trajectories = append(trajectories, run.Trajectory)

		pr := gradePhase(ctx, sb, task, phase, run.Trajectory)
		result.Phases = append(result.Phases, pr)

		if pr.Status == StatusFail {
			result.Status = StatusFail
			result.Reason = fmt.Sprintf("phase %q: %s", phase.Name, pr.Reason)
		}
	}

	result.Metrics = ComputeMetrics(ContextExamples(guide), trajectories...)

	if opts.Judge != nil && task.Rubric != "" && len(trajectories) > 0 {
		rubric, err := opts.Judge.Score(ctx, JudgeRequest{
			Rubric:   task.Rubric,
			Evidence: BuildEvidence(ctx, sb, task, trajectories),
			Dir:      sb.Work,
			Env:      sb.Env(),
		})
		if err != nil {
			return nil, err
		}

		result.Rubric = rubric
	}
	result.BudgetExceeded = result.Metrics.ExceededBudget(task.Budget)

	if len(result.BudgetExceeded) > 0 && result.Status == StatusPass {
		result.Status = StatusFail
		result.Reason = "budget: " + strings.Join(result.BudgetExceeded, ", ")
	}

	return result, nil
}

// gradePhase runs the checks of the phase. If the subject did not finish
// cleanly, the phase fails before any check runs.
func gradePhase(ctx context.Context, sb *Sandbox, task *Task, phase Phase, traj *Trajectory) PhaseResult {
	pr := PhaseResult{Name: phase.Name, Status: StatusPass, Subtype: traj.Subtype}

	if traj.Subtype != "success" {
		pr.Status = StatusFail
		pr.Reason = "subject ended with " + traj.Subtype

		for _, c := range phase.Checks {
			pr.Checks = append(pr.Checks, CheckResult{Name: c.Name, Status: StatusNotRun})
		}

		return pr
	}

	for _, c := range phase.Checks {
		cr := runCheck(ctx, sb, task, c, traj.FinalText)
		pr.Checks = append(pr.Checks, cr)

		if cr.Status == StatusFail && pr.Status == StatusPass {
			pr.Status = StatusFail
			pr.Reason = fmt.Sprintf("check %q: %s", c.Name, cr.Error)
		}
	}

	return pr
}

func runCheck(ctx context.Context, sb *Sandbox, task *Task, c Check, finalText string) CheckResult {
	cr := CheckResult{Name: c.Name, Status: StatusPass}

	var err error

	if c.Run != "" {
		var out CommandOutput

		out, err = sb.Run(ctx, expand(c.Run, task))
		if err == nil {
			err = AssertExpect(c.Expect, out)
		}
	} else {
		err = AssertAnswer(c, finalText)
	}

	if err != nil {
		cr.Status = StatusFail
		cr.Error = err.Error()
	}

	return cr
}

// expand replaces $NS with the task namespace in prompts and check
// commands. Then task.yaml holds the namespace once.
func expand(s string, task *Task) string {
	return strings.ReplaceAll(s, "$NS", task.Namespace)
}

// BuildSystemPrompt returns the system prompt for the subject. It holds the
// sandbox facts and then the context guide, verbatim. It holds nothing else
// about xdb.
func BuildSystemPrompt(sb *Sandbox, guide string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "You are working in the directory %s.\n", sb.Work)

	if len(sb.Fixtures) > 0 {
		fmt.Fprintf(&b, "It contains these files: %s.\n", strings.Join(sb.Fixtures, ", "))
	}

	b.WriteString("The `xdb` command-line tool is installed and its daemon is running.\n")
	b.WriteString("Do not ask questions. When a decision is needed, make it and state it in your reply.\n")
	b.WriteString("When the task asks for an ANSWER, end your reply with one line of the form `ANSWER: <value>` and nothing after it.\n")
	b.WriteString("\n")
	b.WriteString(guide)

	return b.String()
}

func writeRaw(dir string, index int, phase string, raw []byte) error {
	if dir == "" {
		return nil
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("[xdb/evals] create results dir: %w", err)
	}

	name := fmt.Sprintf("trajectory.%02d-%s.jsonl", index+1, phase)
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
		return fmt.Errorf("[xdb/evals] write trajectory: %w", err)
	}

	return nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("[xdb/evals] create results dir: %w", err)
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("[xdb/evals] encode %s: %w", filepath.Base(path), err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("[xdb/evals] write %s: %w", filepath.Base(path), err)
	}

	return nil
}
