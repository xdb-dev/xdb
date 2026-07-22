package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillsGet_UnknownSkill_NotFound(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", configPath, "skills", "get", "nosuch")

	assert.Equal(t, ExitAppError, code)
	assert.Empty(t, stdout)

	var env describeErrEnvelope
	require.NoError(t, json.Unmarshal([]byte(stderr), &env), "stderr: %s", stderr)
	assert.Equal(t, CodeNotFound, env.Code)
	assert.Equal(t, "skills", env.Resource)
	assert.Equal(t, "get", env.Action)
	assert.Contains(t, env.Hint, "xdb skills")
}

func TestSkillsGet_MissingArg_InvalidArgument(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", configPath, "skills", "get")

	assert.Equal(t, ExitInvalidArgs, code)
	assert.Empty(t, stdout)

	var env describeErrEnvelope
	require.NoError(t, json.Unmarshal([]byte(stderr), &env), "stderr: %s", stderr)
	assert.Equal(t, CodeInvalidArgument, env.Code)
	assert.Equal(t, "skills", env.Resource)
	assert.Equal(t, "get", env.Action)
}

func TestSkillsGet_KnownSkill_PrintsContent(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, _, code := runCLI(t, "--config", configPath, "skills", "get", "getting-started")

	assert.Equal(t, ExitOK, code)
	assert.NotEmpty(t, stdout)
}

func TestSkillsBareNameActsAsGet(t *testing.T) {
	cfg, _ := tempCLIConfig(t)

	stdout, _, code := runCLI(t, "--config", cfg, "skills", "getting-started")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, "# Getting Started")
	assert.NotContains(t, stdout, `"category"`, "must render the doc, not the list")
}

func TestSkillsBareUnknownName(t *testing.T) {
	cfg, _ := tempCLIConfig(t)

	_, stderr, code := runCLI(t, "--config", cfg, "skills", "nope")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "NOT_FOUND")
}

func TestSkillsList_AllFourPresent(t *testing.T) {
	cfg, _ := tempCLIConfig(t)

	stdout, _, code := runCLI(t, "--config", cfg, "skills", "-o", "json")
	require.Equal(t, 0, code)

	for _, name := range []string{"getting-started", "query-and-filter", "schema-evolution", "bulk-data"} {
		assert.Contains(t, stdout, name)
	}
}

func TestGettingStartedContent(t *testing.T) {
	cfg, _ := tempCLIConfig(t)

	stdout, _, code := runCLI(t, "--config", cfg, "skills", "get", "getting-started")
	require.Equal(t, 0, code)

	assert.Contains(t, stdout, `"boolean"`)
	assert.Contains(t, stdout, `"fields"`)
	assert.Contains(t, stdout, "xdb init")
	assert.Contains(t, stdout, "Next steps")
	assert.NotContains(t, stdout, `"Type"`)
	assert.NotContains(t, stdout, `"bool"}`)
}

// TestSkillsPayloadsAreValidAgainstServer replays every fenced bash
// command in every skill doc that carries a --json payload, proving
// the docs can never drift from what the server accepts.
func TestSkillsPayloadsAreValidAgainstServer(t *testing.T) {
	cfg := startCLITestDaemon(t)

	// Skills build on getting-started's schema, so replay in reading
	// order and re-seed the record it deletes at its end.
	order := []string{"getting-started", "schema-evolution", "query-and-filter", "bulk-data"}
	require.Len(t, skills, len(order), "new skills must be added to the replay order")

	seed := func() {
		_, _, seedCode := runCLI(t, "--config", cfg, "records", "upsert",
			"--uri", "xdb://myapp/todos/todo-1",
			"--json", `{"title":"Try XDB","done":false}`, "--quiet")
		require.Equal(t, 0, seedCode)
	}

	for _, name := range order {
		s, ok := skills[name]
		require.True(t, ok, "skill %s missing", name)

		if name != "getting-started" {
			seed()
		}

		blocks := regexp.MustCompile("(?s)```bash\n(.*?)```").FindAllStringSubmatch(s.Content, -1)
		require.NotEmpty(t, blocks, "skill %s has no bash blocks", name)

		for _, b := range blocks {
			for _, command := range splitSkillCommands(b[1]) {
				if !strings.HasPrefix(command, "xdb ") {
					continue
				}
				// Skip commands needing stdin, shell features, or
				// live state the replay can't know (revision CAS).
				if strings.ContainsAny(command, "<>|$") || strings.Contains(command, `"revision"`) {
					continue
				}

				args, argErr := shellSplit(strings.TrimPrefix(command, "xdb "))
				require.NoError(t, argErr, "skill %s command %q", name, command)

				_, stderr, code := runCLI(t, append([]string{"--config", cfg}, args...)...)
				assert.Equal(t, 0, code, "skill %s: %q failed: %s", name, command, stderr)
			}
		}
	}
}

// splitSkillCommands joins continuation lines of multi-line commands.
func splitSkillCommands(block string) []string {
	var commands []string
	var current strings.Builder
	depth := 0

	for line := range strings.SplitSeq(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && depth == 0 {
			continue
		}

		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(trimmed)

		depth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
		depth += strings.Count(trimmed, "[") - strings.Count(trimmed, "]")
		if depth == 0 {
			commands = append(commands, current.String())
			current.Reset()
		}
	}

	return commands
}

// shellSplit splits a command line honoring single quotes.
func shellSplit(s string) ([]string, error) {
	var args []string
	var current strings.Builder
	inSingle := false

	for _, r := range s {
		switch {
		case r == '\'':
			inSingle = !inSingle
		case r == ' ' && !inSingle:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if inSingle {
		return nil, fmt.Errorf("unterminated quote in %q", s)
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args, nil
}
