package evals

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// CommandOutput is what a Run check captured.
type CommandOutput struct {
	Exit   int
	Stdout string
	Stderr string
}

// AssertExpect applies every set assertion in e to out and returns the
// first failure.
func AssertExpect(e Expect, out CommandOutput) error {
	if e.Exit != nil && out.Exit != *e.Exit {
		return fmt.Errorf("exit: want %d, got %d; stderr: %s", *e.Exit, out.Exit, truncate(out.Stderr))
	}

	if e.ExitNonzero && out.Exit == 0 {
		return fmt.Errorf("exit_nonzero: got 0")
	}

	if e.StdoutContains != "" && !strings.Contains(out.Stdout, e.StdoutContains) {
		return fmt.Errorf("stdout_contains: %q not in stdout: %s", e.StdoutContains, truncate(out.Stdout))
	}

	if e.StderrContains != "" && !strings.Contains(out.Stderr, e.StderrContains) {
		return fmt.Errorf("stderr_contains: %q not in stderr: %s", e.StderrContains, truncate(out.Stderr))
	}

	if e.StdoutEmpty && strings.TrimSpace(out.Stdout) != "" {
		return fmt.Errorf("stdout_empty: got %s", truncate(out.Stdout))
	}

	if e.JSON != nil {
		if err := assertJSON(e.JSON, out.Stdout); err != nil {
			return err
		}
	}

	if e.NDJSONCount != nil || e.NDJSONIDs != nil {
		if err := assertNDJSON(e, out.Stdout); err != nil {
			return err
		}
	}

	if e.Error != nil {
		if err := assertError(e.Error, out); err != nil {
			return err
		}
	}

	return nil
}

// AssertAnswer grades the ANSWER line of the subject's final reply.
func AssertAnswer(c Check, finalText string) error {
	got, ok := AnswerLine(finalText)
	if !ok {
		return fmt.Errorf("answer: no ANSWER line in reply: %s", truncate(finalText))
	}

	if c.Answer != nil && got != *c.Answer {
		return fmt.Errorf("answer: want %q, got %q", *c.Answer, got)
	}

	if c.AnswerRegex != "" {
		re, err := regexp.Compile(c.AnswerRegex)
		if err != nil {
			return fmt.Errorf("answer_regex: %w", err)
		}

		if !re.MatchString(got) {
			return fmt.Errorf("answer_regex: %q does not match %q", got, c.AnswerRegex)
		}
	}

	return nil
}

func assertJSON(want map[string]any, stdout string) error {
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		return fmt.Errorf("json: stdout is not a JSON object: %s", truncate(stdout))
	}

	if err := partialMatch(want, got); err != nil {
		return fmt.Errorf("json: %w", err)
	}

	return nil
}

// partialMatch makes sure that every key in want is present in got with an
// equal value. Nested maps recurse. Numbers compare by value, so a YAML 1
// matches a JSON 1.0.
func partialMatch(want, got map[string]any) error {
	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		w := want[k]

		g, ok := got[k]
		if !ok {
			return fmt.Errorf("%s: absent", k)
		}

		wm, wIsMap := w.(map[string]any)
		gm, gIsMap := g.(map[string]any)

		if wIsMap && gIsMap {
			if err := partialMatch(wm, gm); err != nil {
				return fmt.Errorf("%s.%w", k, err)
			}

			continue
		}

		if !valuesEqual(w, g) {
			return fmt.Errorf("%s: want %v, got %v", k, w, g)
		}
	}

	return nil
}

func valuesEqual(want, got any) bool {
	wn, wIsNum := toFloat(want)
	gn, gIsNum := toFloat(got)

	if wIsNum && gIsNum {
		return wn == gn
	}

	return reflect.DeepEqual(want, got)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

func assertNDJSON(e Expect, stdout string) error {
	var (
		count int
		ids   []string
	)

	for i, line := range strings.Split(stdout, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			return fmt.Errorf("ndjson_count: line %d is not JSON: %s", i+1, truncate(line))
		}

		count++

		if id, ok := obj["_id"].(string); ok {
			ids = append(ids, id)
		}
	}

	if e.NDJSONCount != nil && count != *e.NDJSONCount {
		return fmt.Errorf("ndjson_count: want %d, got %d", *e.NDJSONCount, count)
	}

	if e.NDJSONIDs != nil {
		want := append([]string(nil), e.NDJSONIDs...)
		sort.Strings(want)
		sort.Strings(ids)

		if !reflect.DeepEqual(want, ids) {
			return fmt.Errorf("ndjson_ids: want %v, got %v", want, ids)
		}
	}

	return nil
}

// assertError parses the error envelope from stdout. If stdout is empty, it
// parses stderr. Then it does a partial match.
func assertError(want map[string]any, out CommandOutput) error {
	source := out.Stdout
	if strings.TrimSpace(source) == "" {
		source = out.Stderr
	}

	start := strings.Index(source, "{")
	if start < 0 {
		return fmt.Errorf("error: no JSON envelope in output: %s", truncate(source))
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(source[start:]), &got); err != nil {
		return fmt.Errorf("error: no JSON envelope in output: %s", truncate(source))
	}

	if err := partialMatch(want, got); err != nil {
		return fmt.Errorf("error: %w", err)
	}

	return nil
}

const truncateAt = 400

func truncate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= truncateAt {
		return s
	}

	return s[:truncateAt] + "..."
}
