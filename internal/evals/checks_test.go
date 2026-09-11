package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertExit(t *testing.T) {
	require.NoError(t, AssertExit(CommandOutput{Stdout: "ok"}))

	err := AssertExit(CommandOutput{Exit: 1, Stderr: `{"code":"NOT_FOUND"}`})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit 1")
	assert.Contains(t, err.Error(), "NOT_FOUND", "the output says what went wrong")

	err = AssertExit(CommandOutput{Exit: 2})
	require.ErrorContains(t, err, "exit 2")
}

func TestAssertAnswer(t *testing.T) {
	want := func(s string) *string { return &s }

	tests := []struct {
		name    string
		check   Check
		text    string
		wantErr string
	}{
		{"exact", Check{Answer: want("42")}, "ANSWER: 42", ""},
		{"mismatch", Check{Answer: want("42")}, "ANSWER: 41", `want "42", got "41"`},
		{"missing line", Check{Answer: want("42")}, "no answer", "no ANSWER line"},
		{"regex", Check{AnswerRegex: `^4\d$`}, "ANSWER: 42", ""},
		{"regex mismatch", Check{AnswerRegex: `^4\d$`}, "ANSWER: 99", "does not match"},
		{"bad regex", Check{AnswerRegex: `[`}, "ANSWER: 1", "answer_regex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssertAnswer(tt.check, tt.text)

			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "short", truncate("  short  "))

	long := make([]byte, truncateAt+10)
	for i := range long {
		long[i] = 'x'
	}

	assert.Len(t, truncate(string(long)), truncateAt+3)
}
