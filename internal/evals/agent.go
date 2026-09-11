package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3/option"
	"github.com/sonnes/pi-go/pkg/agent"
	"github.com/sonnes/pi-go/pkg/ai"
	"github.com/sonnes/pi-go/pkg/ai/provider/openairesponses"
	"github.com/sonnes/pi-go/pkg/catalog"
)

// ErrBudget is returned by a subject that spent its cost budget. The phase
// fails. The harness does not.
var ErrBudget = errors.New("[xdb/evals] cost budget spent")

// ErrNoAPIKey is returned when OPENROUTER_API_KEY is not in the environment.
var ErrNoAPIKey = errors.New("[xdb/evals] OPENROUTER_API_KEY is not set")

const (
	providerID    = "openrouter"
	openRouterURL = "https://openrouter.ai/api/v1"
)

// models are the subject models the harness knows, by OpenRouter slug.
//
// The catalog is explicit because auto-detection registers OpenRouter under
// the OpenAI model list, where an OpenRouter slug does not resolve. Cost is
// US dollars per million tokens, from openrouter.ai. A model with no cost
// reports a cost of zero, and the cost budget then never trips.
//
// To add a model, add a row.
var models = []ai.Model{
	{
		ID:               "anthropic/claude-haiku-4.5",
		ToolCall:         true,
		StructuredOutput: true,
		ContextWindow:    200000,
		MaxTokens:        64000,
		Cost:             ai.Cost{Input: 1, Output: 5, CacheRead: 0.1, CacheWrite: 1.25},
	},
	{
		ID:               "anthropic/claude-sonnet-4.6",
		ToolCall:         true,
		StructuredOutput: true,
		ContextWindow:    200000,
		MaxTokens:        64000,
		Cost:             ai.Cost{Input: 3, Output: 15, CacheRead: 0.3, CacheWrite: 3.75},
	},
}

// Subject is the agent under test. The harness opens one per task and calls
// Run once per phase, in order. The conversation carries across the phases.
type Subject interface {
	Run(ctx context.Context, prompt string) (*Trace, error)
	Close() error
}

// SubjectSpec is what a subject needs to run a task.
type SubjectSpec struct {
	Model        string
	SystemPrompt string
	MaxTurns     int
	// BudgetUSD stops the run once the spend passes it. Zero is no limit.
	BudgetUSD float64
}

// OpenSubject opens the subject for one task in a sandbox.
type OpenSubject func(sb *Sandbox, spec SubjectSpec) (Subject, error)

// OpenRouterSubject runs the subject on OpenRouter, in this process. Its
// only tool is a shell that runs inside the sandbox.
func OpenRouterSubject(sb *Sandbox, spec SubjectSpec) (Subject, error) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		return nil, ErrNoAPIKey
	}

	provider := openairesponses.NewForOpenRouter(
		option.WithAPIKey(key),
		option.WithBaseURL(openRouterURL),
	)

	cat := catalog.New()
	cat.RegisterTextProvider(providerID, provider, models...)

	s := &piSubject{maxTurns: spec.MaxTurns, budget: spec.BudgetUSD}

	a, err := cat.Agent(providerID+"/"+spec.Model,
		agent.WithSystemPrompt(spec.SystemPrompt),
		agent.WithTools(shellTool(sb)),
		agent.WithMaxTurns(spec.MaxTurns),
		agent.WithAfterTurn(s.chargeTurn),
	)
	if err != nil {
		return nil, fmt.Errorf("[xdb/evals] open subject: %w", err)
	}

	s.agent = a

	return s, nil
}

// piSubject holds one conversation and the spend so far.
type piSubject struct {
	agent    agent.Agent
	maxTurns int
	budget   float64
	spent    float64
}

// Run runs one phase and collects its trace. A budget stop returns the
// partial trace with [ErrBudget], so the caller can still grade the phase.
func (s *piSubject) Run(ctx context.Context, prompt string) (*Trace, error) {
	tr := newTrace(s.maxTurns)

	stream := s.agent.Run(ctx, ai.UserMessage(prompt))
	for ev, err := range stream.Events() {
		if err != nil {
			continue // Wait reports it.
		}

		tr.add(ev)
	}

	if _, err := stream.Wait(); err != nil {
		return tr, err
	}

	return tr, nil
}

func (s *piSubject) Close() error { return s.agent.Close() }

// chargeTurn adds the cost of a turn and stops the run once the budget is
// spent. pi has no cost limit of its own.
func (s *piSubject) chargeTurn(
	_ context.Context,
	turn agent.TurnResult,
	_ []ai.Message,
) ([]ai.Message, error) {
	s.spent += totalCost(turn.Usage)

	if s.budget > 0 && s.spent > s.budget {
		return nil, fmt.Errorf("%w: $%.2f of $%.2f", ErrBudget, s.spent, s.budget)
	}

	return nil, nil
}

// shellToolName is the name the subject calls the shell by. [ComputeMetrics]
// reads it back off the trace.
const shellToolName = "bash"

// shellInput is the argument of the shell tool. The field name matches the
// name that [Trace.add] reads back out of the call.
type shellInput struct {
	Command string `json:"command" jsonschema:"The shell command to run."`
}

// shellDescription tells the subject what the shell does. It does not
// promise a working directory that survives a call, because every command
// runs in the work directory of the sandbox.
const shellDescription = `Run a shell command in the working directory.

Each command runs on its own. A cd does not carry into the next call, so use
absolute paths. Chain dependent commands with && in one call. A command that
exits non-zero comes back as an error with its output.`

// shellTool gives the subject a shell inside the sandbox. A non-zero exit
// is a tool error, so the model sees the failure and the trace records it.
func shellTool(sb *Sandbox) ai.Tool {
	return ai.DefineTool(shellToolName, shellDescription,
		func(ctx context.Context, in shellInput) (string, error) {
			out, err := sb.Run(ctx, in.Command)
			if err != nil {
				return "", err
			}

			text := out.Stdout + out.Stderr

			if out.Exit != 0 {
				return "", fmt.Errorf("Exit code %d\n%s", out.Exit, text) //nolint:staticcheck // the model reads this text
			}

			return text, nil
		})
}
