package evals

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// PhaseRequest is everything a Runner needs to run one phase of a task.
type PhaseRequest struct {
	Prompt       string
	SystemPrompt string
	Model        string
	MaxTurns     int
	MaxBudgetUSD float64
	// If SessionID is set, the runner resumes the conversation of an
	// earlier phase.
	SessionID string
	Dir       string
	Env       []string
}

// PhaseRun is the outcome of one phase. It holds the parsed trajectory and
// the raw stream for the results directory.
type PhaseRun struct {
	Trajectory *Trajectory
	Raw        []byte
	Stderr     string
}

// Runner drives the subject agent for one phase. The harness ships a
// [ClaudeRunner]. Tests use an in-memory runner.
type Runner interface {
	RunPhase(ctx context.Context, req PhaseRequest) (*PhaseRun, error)
}

// ClaudeRunner runs the subject with headless Claude Code. The prompt goes
// in on stdin. The runner disables the settings, hooks, and skills of the
// user. The subject knows only what the system prompt tells it.
type ClaudeRunner struct {
	// Binary is the claude executable. Empty means "claude" on PATH.
	Binary string
}

// Args returns the command-line arguments for a phase request.
func (r ClaudeRunner) Args(req PhaseRequest) []string {
	args := []string{
		"-p",
		"--model", req.Model,
		"--output-format", "stream-json",
		"--verbose",
		"--allowedTools", "Bash,Read,Write",
		"--max-turns", strconv.Itoa(req.MaxTurns),
		"--setting-sources", "",
		"--disable-slash-commands",
		"--permission-mode", "bypassPermissions",
		"--append-system-prompt", req.SystemPrompt,
	}

	if req.MaxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", strconv.FormatFloat(req.MaxBudgetUSD, 'f', 2, 64))
	}

	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}

	return args
}

// RunPhase runs claude and parses its trajectory. A non-zero exit code is
// not an error by itself. A max-turns stop still gives a result record, and
// the caller grades the subtype.
func (r ClaudeRunner) RunPhase(ctx context.Context, req PhaseRequest) (*PhaseRun, error) {
	binary := r.Binary
	if binary == "" {
		binary = "claude"
	}

	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, binary, r.Args(req)...)
	cmd.Dir = req.Dir
	cmd.Env = req.Env
	cmd.Stdin = strings.NewReader(req.Prompt)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	var exitErr *exec.ExitError
	if runErr != nil && !errors.As(runErr, &exitErr) {
		return nil, fmt.Errorf("[xdb/evals] run claude: %w", runErr)
	}

	traj, err := ParseTrajectory(bytes.NewReader(stdout.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("%w; claude stderr: %s", err, truncate(stderr.String()))
	}

	return &PhaseRun{
		Trajectory: traj,
		Raw:        stdout.Bytes(),
		Stderr:     stderr.String(),
	}, nil
}
