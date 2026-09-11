package evals

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// ErrNoResult is the error for a trajectory stream that ends without a
// result record. The subject stopped before it finished.
var ErrNoResult = errors.New("[xdb/evals] trajectory has no result record")

// Trajectory is the parsed stream-json output of one subject phase.
type Trajectory struct {
	SessionID string
	// Subtype is the result subtype: "success", "error_max_turns", and so on.
	Subtype string
	// FinalText is the subject's last reply. Answer checks read it.
	FinalText    string
	NumTurns     int
	DurationMS   int64
	CostUSD      float64
	InputTokens  int
	OutputTokens int
	// Calls lists every tool call in issue order. The parser links each
	// result to its call by id.
	Calls []Call
}

// Call is one tool call and its result.
type Call struct {
	ID   string
	Tool string
	// Command is the shell command for a Bash call, empty otherwise.
	Command string
	// Output is the text the tool returned.
	Output string
	// Failed is true when the tool reported an error. For Bash, that is a
	// non-zero exit code.
	Failed bool
}

type streamRecord struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype"`
	SessionID string          `json:"session_id"`
	Message   json.RawMessage `json:"message"`

	// Result fields.
	Result     string  `json:"result"`
	NumTurns   int     `json:"num_turns"`
	DurationMS int64   `json:"duration_ms"`
	CostUSD    float64 `json:"total_cost_usd"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type streamMessage struct {
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     map[string]any  `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
	Text      string          `json:"text"`
}

// ParseTrajectory reads a stream-json trajectory from
// `claude -p --output-format stream-json --verbose`.
func ParseTrajectory(r io.Reader) (*Trajectory, error) {
	traj := &Trajectory{}
	byID := map[string]int{}
	sawResult := false

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1<<20), 64<<20)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var rec streamRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("[xdb/evals] parse trajectory: %w", err)
		}

		if rec.SessionID != "" && traj.SessionID == "" {
			traj.SessionID = rec.SessionID
		}

		switch rec.Type {
		case "assistant":
			if err := traj.addToolUses(rec.Message, byID); err != nil {
				return nil, err
			}
		case "user":
			if err := traj.addToolResults(rec.Message, byID); err != nil {
				return nil, err
			}
		case "result":
			sawResult = true
			traj.Subtype = rec.Subtype
			traj.FinalText = rec.Result
			traj.NumTurns = rec.NumTurns
			traj.DurationMS = rec.DurationMS
			traj.CostUSD = rec.CostUSD
			traj.InputTokens = rec.Usage.InputTokens
			traj.OutputTokens = rec.Usage.OutputTokens
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[xdb/evals] read trajectory: %w", err)
	}

	if !sawResult {
		return nil, ErrNoResult
	}

	return traj, nil
}

func (t *Trajectory) addToolUses(raw json.RawMessage, byID map[string]int) error {
	blocks, err := parseBlocks(raw)
	if err != nil {
		return err
	}

	for _, b := range blocks {
		if b.Type != "tool_use" {
			continue
		}

		call := Call{ID: b.ID, Tool: b.Name}
		if cmd, ok := b.Input["command"].(string); ok {
			call.Command = cmd
		}

		byID[b.ID] = len(t.Calls)
		t.Calls = append(t.Calls, call)
	}

	return nil
}

func (t *Trajectory) addToolResults(raw json.RawMessage, byID map[string]int) error {
	blocks, err := parseBlocks(raw)
	if err != nil {
		return err
	}

	for _, b := range blocks {
		if b.Type != "tool_result" {
			continue
		}

		idx, ok := byID[b.ToolUseID]
		if !ok {
			continue
		}

		t.Calls[idx].Output = blockText(b.Content)
		t.Calls[idx].Failed = b.IsError
	}

	return nil
}

// parseBlocks returns the content blocks of a message. A plain-string
// content, such as a user prompt, gives no blocks.
func parseBlocks(raw json.RawMessage) ([]contentBlock, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var msg streamMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("[xdb/evals] parse message: %w", err)
	}

	if len(msg.Content) == 0 || msg.Content[0] != '[' {
		return nil, nil
	}

	var blocks []contentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil, fmt.Errorf("[xdb/evals] parse content: %w", err)
	}

	return blocks, nil
}

// blockText flattens a tool_result content. The content is a string or a
// list of text blocks.
func blockText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}

		return ""
	}

	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == "text" {
			parts = append(parts, b.Text)
		}
	}

	return strings.Join(parts, "\n")
}

var answerRe = regexp.MustCompile(`ANSWER:\s*(.*)`)

// AnswerLine extracts the value of the last `ANSWER:` line in text. It
// strips Markdown emphasis and backticks around the value.
func AnswerLine(text string) (string, bool) {
	lines := strings.Split(text, "\n")

	for i := len(lines) - 1; i >= 0; i-- {
		m := answerRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}

		value := strings.Trim(m[1], " \t*`_")

		return value, true
	}

	return "", false
}
