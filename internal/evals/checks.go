package main

import (
	"fmt"
	"regexp"
	"strings"
)

// CommandOutput is the result of one shell command.
type CommandOutput struct {
	Exit   int
	Stdout string
	Stderr string
}

// AssertExit grades a run check. Exit code 0 passes.
//
// The shell already counts, compares, and reads JSON, so a check says what
// it wants in the command itself. The captured output rides along on a
// failure, because an exit code alone does not say what went wrong.
func AssertExit(out CommandOutput) error {
	if out.Exit == 0 {
		return nil
	}

	text := truncate(out.Stdout + "\n" + out.Stderr)
	if text == "" {
		return fmt.Errorf("exit %d", out.Exit)
	}

	return fmt.Errorf("exit %d: %s", out.Exit, text)
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

const truncateAt = 400

func truncate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= truncateAt {
		return s
	}

	return s[:truncateAt] + "..."
}
