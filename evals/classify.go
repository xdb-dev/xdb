package evals

import (
	"regexp"
	"strings"
)

// Layer is a progressive-disclosure layer of the xdb CLI. Layer 0 is the
// context guide in the system prompt. The agent must ask for the higher
// layers.
type Layer int

const (
	// LayerNone marks a call that is not a discovery call.
	LayerNone Layer = -1
	// LayerContext is `xdb context`, the guide already in the system prompt.
	LayerContext Layer = 0
	// LayerOverview is root help, resource help, and `describe --actions`.
	LayerOverview Layer = 1
	// LayerAction is action help and `describe <resource>.<action>`.
	LayerAction Layer = 2
	// LayerReference is `describe --uri`, `--filter`, `--errors`, and the
	// other reference topics.
	LayerReference Layer = 3
	// LayerSkills is `xdb skills` and `xdb skills get`.
	LayerSkills Layer = 4
)

// XDBCall is the classification of one xdb invocation inside a shell
// command. Resource and Action name what the call does or asks about.
type XDBCall struct {
	Resource  string
	Action    string
	Discovery bool
	Layer     Layer
}

// Key returns "resource action", the key used to match a call against
// prior discovery and context examples.
func (c XDBCall) Key() string {
	return strings.TrimSpace(c.Resource + " " + c.Action)
}

var rootAliases = map[string][2]string{
	"get":         {"records", "get"},
	"put":         {"records", "upsert"},
	"ls":          {"records", "list"},
	"rm":          {"records", "delete"},
	"make-schema": {"schemas", "create"},
}

var rootFlagsWithValue = map[string]bool{
	"-c": true, "--config": true, "-o": true, "--output": true,
}

var describeOverviewFlags = map[string]bool{
	"--actions": true, "--methods": true, "--types": true,
}

var describeReferenceFlags = map[string]bool{
	"--uri": true, "--errors": true, "--value-types": true, "--config": true,
	"--daemon": true, "--schema-format": true,
}

// ClassifyCommand finds the first xdb invocation in a shell command and
// classifies it. If the command does not run xdb, it returns false. If a
// command chains several xdb calls, the first call sets the class.
func ClassifyCommand(cmd string) (XDBCall, bool) {
	args, ok := xdbSegment(cmd)
	if !ok {
		return XDBCall{}, false
	}

	args = stripRootFlags(args)
	help := hasHelpFlag(args)
	words, flags := splitWords(args)

	if len(words) == 0 {
		return XDBCall{Discovery: true, Layer: LayerOverview}, true
	}

	resource, action := words[0], ""
	if len(words) > 1 {
		action = words[1]
	}

	if alias, isAlias := rootAliases[resource]; isAlias {
		resource, action = alias[0], alias[1]
	}

	switch resource {
	case "context":
		return XDBCall{Resource: resource, Discovery: true, Layer: LayerContext}, true
	case "help":
		return XDBCall{Discovery: true, Layer: LayerOverview}, true
	case "describe":
		return classifyDescribe(action, flags), true
	case "skills":
		if action == "get" && len(words) > 2 {
			action = words[2]
		} else {
			action = ""
		}

		return XDBCall{Resource: resource, Action: action, Discovery: true, Layer: LayerSkills}, true
	}

	call := XDBCall{Resource: resource, Action: action, Layer: LayerNone}

	if help {
		call.Discovery = true
		call.Layer = LayerOverview

		if action != "" {
			call.Layer = LayerAction
		}
	}

	return call, true
}

func classifyDescribe(positional string, flags []string) XDBCall {
	call := XDBCall{Resource: "describe", Discovery: true, Layer: LayerOverview}

	for _, f := range flags {
		switch {
		case describeOverviewFlags[f]:
			return call
		case f == "--filter":
			call.Resource, call.Action, call.Layer = "records", "list", LayerReference

			return call
		case describeReferenceFlags[f]:
			call.Layer = LayerReference

			return call
		}
	}

	if positional == "" {
		return call
	}

	if res, act, found := strings.Cut(positional, "."); found {
		call.Resource, call.Action, call.Layer = res, act, LayerAction

		return call
	}

	call.Layer = LayerReference

	return call
}

// xdbSegment returns the arguments of the first xdb invocation in cmd. These
// are the tokens after an unquoted `xdb` word, up to the next command
// separator.
func xdbSegment(cmd string) ([]string, bool) {
	tokens := shellTokens(cmd)

	for i, tok := range tokens {
		if tok.sep || tok.text != "xdb" {
			continue
		}

		var args []string

		for _, next := range tokens[i+1:] {
			if next.sep {
				break
			}

			args = append(args, next.text)
		}

		return args, true
	}

	return nil, false
}

func stripRootFlags(args []string) []string {
	for len(args) > 0 && rootFlagsWithValue[args[0]] {
		if len(args) < 2 {
			return nil
		}

		args = args[2:]
	}

	return args
}

func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}

	return false
}

// splitWords separates positional words from flags. It skips the value of
// a flag that takes one, so that `-o json records list` does not read
// "json" as a resource.
func splitWords(args []string) (words, flags []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]

		if !strings.HasPrefix(a, "-") {
			words = append(words, a)

			continue
		}

		if strings.HasPrefix(a, "-") && strings.Contains(a, "=") {
			a, _, _ = strings.Cut(a, "=")
		}

		flags = append(flags, a)

		if i+1 < len(args) && takesValue(a) && !strings.HasPrefix(args[i+1], "-") {
			i++
		}
	}

	return words, flags
}

var valueFlags = map[string]bool{
	"-o": true, "--output": true, "-c": true, "--config": true,
	"--uri": true, "--json": true, "-f": true, "--file": true,
	"--filter": true, "--fields": true, "--limit": true, "--offset": true,
}

func takesValue(flag string) bool {
	return valueFlags[flag]
}

type shellToken struct {
	text string
	sep  bool
}

// shellTokens splits a command into words. Quotes group, backslashes escape,
// and `| & ; ( ) newline backtick` split commands. It finds command
// boundaries only. It is not a shell.
func shellTokens(cmd string) []shellToken {
	var (
		tokens  []shellToken
		current strings.Builder
		inWord  bool
		quote   rune
	)

	flush := func() {
		if inWord {
			tokens = append(tokens, shellToken{text: current.String()})
			current.Reset()
			inWord = false
		}
	}

	runes := []rune(cmd)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		switch {
		case quote != 0:
			switch {
			case r == quote:
				quote = 0
			case r == '\\' && quote == '"' && i+1 < len(runes):
				i++
				current.WriteRune(runes[i])
			default:
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == '\\' && i+1 < len(runes):
			i++
			current.WriteRune(runes[i])
			inWord = true
		case strings.ContainsRune("|&;()\n`", r):
			flush()
			tokens = append(tokens, shellToken{text: string(r), sep: true})
		case r == ' ' || r == '\t' || r == '\r':
			flush()
		default:
			current.WriteRune(r)
			inWord = true
		}
	}

	flush()

	return tokens
}

var exampleRe = regexp.MustCompile(`(?m)^\s*xdb\s+([a-z-]+)\s+([a-z-]+)\b`)

// ContextExamples returns the "resource action" pairs that have an example
// in the context guide. A discovery call on one of these pairs is redundant,
// because layer 0 already covers it.
func ContextExamples(guide string) map[string]bool {
	examples := map[string]bool{}

	for _, m := range exampleRe.FindAllStringSubmatch(guide, -1) {
		call, ok := ClassifyCommand("xdb " + m[1] + " " + m[2])
		if !ok || call.Discovery {
			continue
		}

		examples[call.Key()] = true
	}

	return examples
}
