package evals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// JudgeRequest is what a Judge needs to score a task run against its
// rubric. Evidence is the text that the judge reads. It holds the final
// schemas, a sample of records, and the xdb commands that the subject ran.
type JudgeRequest struct {
	Rubric   string
	Evidence string
	Model    string
	Dir      string
	Env      []string
}

// RubricResult is the verdict of the judge.
type RubricResult struct {
	Scores  []RubricScore `json:"scores"`
	Average float64       `json:"average"`
	CostUSD float64       `json:"cost_usd"`
}

// RubricScore is one rubric line, scored 1 to 5.
type RubricScore struct {
	Criterion string `json:"criterion"`
	Score     int    `json:"score"`
	Note      string `json:"note"`
}

// Judge scores a run against a rubric. The harness ships a [ClaudeJudge].
type Judge interface {
	Score(ctx context.Context, req JudgeRequest) (*RubricResult, error)
}

// DefaultJudgeModel is the model the judge uses when none is set.
const DefaultJudgeModel = "sonnet"

// ClaudeJudge scores with one headless Claude turn and structured output.
// It has no tools. It reads the evidence and answers.
type ClaudeJudge struct {
	Binary string
	Model  string
}

const judgeSchema = `{
  "type": "object",
  "properties": {
    "scores": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "criterion": {"type": "string"},
          "score": {"type": "integer", "minimum": 1, "maximum": 5},
          "note": {"type": "string"}
        },
        "required": ["criterion", "score", "note"]
      }
    }
  },
  "required": ["scores"]
}`

// Args returns the command-line arguments for a judge request.
func (j ClaudeJudge) Args(req JudgeRequest) []string {
	model := req.Model
	if model == "" {
		model = j.Model
	}

	if model == "" {
		model = DefaultJudgeModel
	}

	return []string{
		"-p",
		"--model", model,
		"--output-format", "json",
		"--json-schema", judgeSchema,
		"--restricted",
		"--max-turns", "1",
		"--setting-sources", "",
		"--disable-slash-commands",
	}
}

// Prompt returns the judge prompt for a request.
func (j ClaudeJudge) Prompt(req JudgeRequest) string {
	var b strings.Builder

	b.WriteString("You are grading how well an agent modeled and used a data store through the `xdb` CLI.\n")
	b.WriteString("Score every line of the rubric from 1 (not at all) to 5 (fully), with a one-sentence note that cites the evidence.\n")
	b.WriteString("Grade only what the evidence shows. If the evidence is silent on a line, score 1 and say so.\n\n")
	b.WriteString("# Rubric\n\n")
	b.WriteString(req.Rubric)
	b.WriteString("\n\n# Evidence\n\n")
	b.WriteString(req.Evidence)

	return b.String()
}

// Score runs the judge and parses its structured output.
func (j ClaudeJudge) Score(ctx context.Context, req JudgeRequest) (*RubricResult, error) {
	binary := j.Binary
	if binary == "" {
		binary = "claude"
	}

	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, binary, j.Args(req)...)
	cmd.Dir = req.Dir
	cmd.Env = req.Env
	cmd.Stdin = strings.NewReader(j.Prompt(req))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	var exitErr *exec.ExitError
	if runErr != nil && !errors.As(runErr, &exitErr) {
		return nil, fmt.Errorf("[xdb/evals] run judge: %w", runErr)
	}

	return parseJudgeOutput(stdout.Bytes(), stderr.String())
}

func parseJudgeOutput(out []byte, stderr string) (*RubricResult, error) {
	var envelope struct {
		Subtype          string          `json:"subtype"`
		CostUSD          float64         `json:"total_cost_usd"`
		StructuredOutput json.RawMessage `json:"structured_output"`
		Result           string          `json:"result"`
	}

	if err := json.Unmarshal(out, &envelope); err != nil {
		return nil, fmt.Errorf("[xdb/evals] parse judge output: %w; stderr: %s", err, truncate(stderr))
	}

	raw := envelope.StructuredOutput
	if len(raw) == 0 || string(raw) == "null" {
		raw = json.RawMessage(envelope.Result)
	}

	var result RubricResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("[xdb/evals] judge returned no scores (%s): %w", envelope.Subtype, err)
	}

	if len(result.Scores) == 0 {
		return nil, fmt.Errorf("[xdb/evals] judge returned no scores (%s)", envelope.Subtype)
	}

	total := 0
	for _, s := range result.Scores {
		total += s.Score
	}

	result.Average = float64(total) / float64(len(result.Scores))
	result.CostUSD = envelope.CostUSD

	return &result, nil
}

// BuildEvidence collects the evidence for the judge. It holds every schema
// in the task namespace, up to three records of each schema, and the xdb
// commands that the subject ran, in order.
func BuildEvidence(ctx context.Context, sb *Sandbox, task *Task, phases []*Trajectory) string {
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
