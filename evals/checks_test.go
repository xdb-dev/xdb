package evals

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func intp(i int) *int { return &i }

func TestAssertExpect(t *testing.T) {
	tests := []struct {
		name   string
		expect Expect
		out    CommandOutput
		want   string // empty means pass
	}{
		{name: "exit ok", expect: Expect{Exit: intp(0)}, out: CommandOutput{Exit: 0}},
		{name: "exit mismatch", expect: Expect{Exit: intp(0)}, out: CommandOutput{Exit: 1, Stderr: "boom"}, want: "exit: want 0, got 1"},
		{name: "nonzero ok", expect: Expect{ExitNonzero: true}, out: CommandOutput{Exit: 3}},
		{name: "nonzero fails on zero", expect: Expect{ExitNonzero: true}, out: CommandOutput{Exit: 0}, want: "exit_nonzero: got 0"},
		{name: "stdout contains", expect: Expect{StdoutContains: "hello"}, out: CommandOutput{Stdout: "say hello"}},
		{name: "stdout missing", expect: Expect{StdoutContains: "hello"}, out: CommandOutput{Stdout: "bye"}, want: `stdout_contains: "hello" not in stdout`},
		{name: "stderr contains", expect: Expect{StderrContains: "force"}, out: CommandOutput{Stderr: "needs --force"}},
		{name: "stdout empty", expect: Expect{StdoutEmpty: true}, out: CommandOutput{Stdout: " \n"}},
		{name: "stdout not empty", expect: Expect{StdoutEmpty: true}, out: CommandOutput{Stdout: "x"}, want: "stdout_empty"},
		{
			name:   "json partial match",
			expect: Expect{JSON: map[string]any{"_id": "a", "n": 1, "nested": map[string]any{"k": true}}},
			out:    CommandOutput{Stdout: `{"_id":"a","n":1,"extra":2,"nested":{"k":true,"j":0}}`},
		},
		{
			name:   "json value mismatch",
			expect: Expect{JSON: map[string]any{"n": 2}},
			out:    CommandOutput{Stdout: `{"n":1}`},
			want:   "json: n: want 2, got 1",
		},
		{
			name:   "json missing key",
			expect: Expect{JSON: map[string]any{"missing": 1}},
			out:    CommandOutput{Stdout: `{"n":1}`},
			want:   "json: missing: absent",
		},
		{
			name:   "json not json",
			expect: Expect{JSON: map[string]any{"n": 1}},
			out:    CommandOutput{Stdout: `nope`},
			want:   "json: stdout is not a JSON object",
		},
		{
			name:   "json float equals int",
			expect: Expect{JSON: map[string]any{"n": 42}},
			out:    CommandOutput{Stdout: `{"n":42.0}`},
		},
		{name: "ndjson count", expect: Expect{NDJSONCount: intp(2)}, out: CommandOutput{Stdout: "{\"_id\":\"a\"}\n{\"_id\":\"b\"}\n\n"}},
		{name: "ndjson count zero", expect: Expect{NDJSONCount: intp(0)}, out: CommandOutput{Stdout: ""}},
		{name: "ndjson count mismatch", expect: Expect{NDJSONCount: intp(1)}, out: CommandOutput{Stdout: "{}\n{}\n"}, want: "ndjson_count: want 1, got 2"},
		{name: "ndjson bad line", expect: Expect{NDJSONCount: intp(1)}, out: CommandOutput{Stdout: "nope\n"}, want: "ndjson_count: line 1 is not JSON"},
		{name: "ndjson ids unordered", expect: Expect{NDJSONIDs: []string{"b", "a"}}, out: CommandOutput{Stdout: "{\"_id\":\"a\"}\n{\"_id\":\"b\"}\n"}},
		{name: "ndjson ids mismatch", expect: Expect{NDJSONIDs: []string{"a"}}, out: CommandOutput{Stdout: "{\"_id\":\"a\"}\n{\"_id\":\"b\"}\n"}, want: "ndjson_ids: want [a], got [a b]"},
		{
			name:   "error envelope on stderr",
			expect: Expect{Error: map[string]any{"code": "NOT_FOUND", "resource": "records", "action": "get"}},
			out:    CommandOutput{Exit: 1, Stderr: `{"code":"NOT_FOUND","resource":"records","action":"get","hint":"x"}`},
		},
		{
			name:   "error envelope on stdout wins",
			expect: Expect{Error: map[string]any{"code": "CONFLICT"}},
			out:    CommandOutput{Exit: 1, Stdout: `{"code":"CONFLICT"}`, Stderr: `{"code":"NOT_FOUND"}`},
		},
		{
			name:   "error code mismatch",
			expect: Expect{Error: map[string]any{"code": "NOT_FOUND"}},
			out:    CommandOutput{Exit: 4, Stderr: `{"code":"INTERNAL"}`},
			want:   "error: code: want NOT_FOUND, got INTERNAL",
		},
		{
			name:   "error not an envelope",
			expect: Expect{Error: map[string]any{"code": "NOT_FOUND"}},
			out:    CommandOutput{Exit: 1, Stderr: `error: something`},
			want:   "error: no JSON envelope",
		},
		{
			name:   "first failing assertion reported",
			expect: Expect{Exit: intp(0), StdoutContains: "x"},
			out:    CommandOutput{Exit: 1},
			want:   "exit: want 0, got 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssertExpect(tt.expect, tt.out)
			if tt.want == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestAssertAnswer(t *testing.T) {
	tests := []struct {
		name  string
		check Check
		text  string
		want  string
	}{
		{name: "exact", check: Check{Answer: strp("4880000")}, text: "done\nANSWER: 4880000"},
		{name: "mismatch", check: Check{Answer: strp("1")}, text: "ANSWER: 2", want: "answer: want \"1\", got \"2\""},
		{name: "missing", check: Check{Answer: strp("1")}, text: "no line", want: "answer: no ANSWER line"},
		{name: "regex", check: Check{AnswerRegex: `^\d+$`}, text: "ANSWER: 42"},
		{name: "regex mismatch", check: Check{AnswerRegex: `^\d+$`}, text: "ANSWER: forty", want: "answer_regex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssertAnswer(tt.check, tt.text)
			if tt.want == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func strp(s string) *string { return &s }
