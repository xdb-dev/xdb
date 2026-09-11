package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v3/option"
	"github.com/sonnes/pi-go/pkg/ai"
	"github.com/sonnes/pi-go/pkg/ai/provider/openairesponses"
	"github.com/sonnes/pi-go/pkg/catalog"
)

// JudgeRequest is what a Judge needs to score a task run against its
// rubric. Evidence is the text that the judge reads. It holds the final
// schemas, a sample of records, and the xdb commands that the subject ran.
type JudgeRequest struct {
	Rubric   string
	Evidence string
	Model    string
}

// RubricResult is the verdict of the judge.
type RubricResult struct {
	Scores  []RubricScore `json:"scores"`
	Average float64       `json:"average"`
	CostUSD float64       `json:"cost_usd"`
}

// RubricScore is one rubric line, scored 1 to 5.
type RubricScore struct {
	Criterion string `json:"criterion" jsonschema:"The rubric line, copied."`
	Score     int    `json:"score" jsonschema:"A score from 1 to 5."`
	Note      string `json:"note" jsonschema:"One sentence citing the evidence."`
}

// Judge scores a run against a rubric.
type Judge interface {
	Score(ctx context.Context, req JudgeRequest) (*RubricResult, error)
}

// DefaultJudgeModel is the model the judge uses when none is set.
const DefaultJudgeModel = "anthropic/claude-sonnet-4.6"

// OpenRouterJudge scores with one structured-output call. It has no tools.
// It reads the evidence the harness collected and answers.
//
// The evidence is fixed, so two runs of the same task are scored from the
// same material and their scores compare.
type OpenRouterJudge struct {
	Model string
}

// scores is the structured output of the judge.
type scores struct {
	Scores []RubricScore `json:"scores" jsonschema:"One entry per rubric line."`
}

// Prompt returns the judge prompt for a request.
func (j OpenRouterJudge) Prompt(req JudgeRequest) ai.Prompt {
	var b strings.Builder

	b.WriteString("# Rubric\n\n")
	b.WriteString(req.Rubric)
	b.WriteString("\n\n# Evidence\n\n")
	b.WriteString(req.Evidence)

	return ai.Prompt{
		System: "You are grading how well an agent modeled and used a data store through the `xdb` CLI.\n" +
			"Score every line of the rubric from 1 (not at all) to 5 (fully), with a one-sentence note that cites the evidence.\n" +
			"Grade only what the evidence shows. If the evidence is silent on a line, score 1 and say so.",
		Messages: []ai.Message{ai.UserMessage(b.String())},
	}
}

// Score asks the model for one rubric verdict and averages the scores.
func (j OpenRouterJudge) Score(ctx context.Context, req JudgeRequest) (*RubricResult, error) {
	model := req.Model
	if model == "" {
		model = j.Model
	}

	if model == "" {
		model = DefaultJudgeModel
	}

	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		return nil, ErrNoAPIKey
	}

	cat := catalog.New()
	cat.RegisterTextProvider(providerID, openairesponses.NewForOpenRouter(
		option.WithAPIKey(key),
		option.WithBaseURL(openRouterURL),
	), models...)

	out, err := catalog.GenerateObject[scores](ctx, cat, providerID+"/"+model, j.Prompt(req))
	if err != nil {
		return nil, fmt.Errorf("[xdb/evals] judge: %w", err)
	}

	if len(out.Object.Scores) == 0 {
		return nil, fmt.Errorf("[xdb/evals] judge returned no scores")
	}

	total := 0
	for _, s := range out.Object.Scores {
		total += s.Score
	}

	return &RubricResult{
		Scores:  out.Object.Scores,
		Average: float64(total) / float64(len(out.Object.Scores)),
		CostUSD: totalCost(out.Usage),
	}, nil
}

// BuildEvidence collects the evidence for the judge. It holds every schema
// in the task namespace, up to three records of each schema, and the xdb
// commands that the subject ran, in order.
func BuildEvidence(ctx context.Context, sb *Sandbox, task *Task, phases []*Trace) string {
	var b strings.Builder

	b.WriteString("## Schemas\n\n")

	names := schemaNames(ctx, sb, task.Namespace)
	for _, name := range names {
		uri := fmt.Sprintf("xdb://%s/%s", task.Namespace, name)

		out, _ := sb.Run(ctx, "xdb schemas get "+uri+" -o json")
		fmt.Fprintf(&b, "### %s\n\n```json\n%s\n```\n\n", uri, strings.TrimSpace(out.Stdout))

		out, _ = sb.Run(ctx, "xdb records list "+uri+" --limit 3 -o ndjson")
		fmt.Fprintf(&b, "Sample records:\n\n```\n%s\n```\n\n", strings.TrimSpace(out.Stdout))
	}

	if len(names) == 0 {
		b.WriteString("(no schemas found in the namespace)\n\n")
	}

	b.WriteString("## Commands the agent ran\n\n```\n")

	for i, traj := range phases {
		fmt.Fprintf(&b, "# phase %d\n", i+1)

		for _, call := range traj.Calls {
			if call.Tool != "Bash" {
				continue
			}

			if _, ok := ClassifyCommand(call.Command); !ok {
				continue
			}

			status := "ok"
			if call.Failed {
				status = "FAILED"
			}

			fmt.Fprintf(&b, "[%s] %s\n", status, truncate(call.Command))
		}
	}

	b.WriteString("```\n")

	return b.String()
}

// schemaNames lists the schemas in a namespace. Each item of the NDJSON
// list has the schema URI. The name is the last path segment of the URI.
func schemaNames(ctx context.Context, sb *Sandbox, namespace string) []string {
	out, err := sb.Run(ctx, "xdb schemas list xdb://"+namespace+" -o ndjson --limit 100")
	if err != nil || out.Exit != 0 {
		return nil
	}

	var names []string

	for _, line := range strings.Split(out.Stdout, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var item struct {
			URI string `json:"uri"`
		}

		if json.Unmarshal([]byte(line), &item) != nil || item.URI == "" {
			continue
		}

		if idx := strings.LastIndex(item.URI, "/"); idx >= 0 {
			names = append(names, item.URI[idx+1:])
		}
	}

	return names
}
