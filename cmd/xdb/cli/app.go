// Package cli implements the xdb command-line interface.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	_ "embed"

	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/validate"
	"github.com/xdb-dev/xdb/rpc/client"
)

//go:embed CONTEXT.md
var agentContext string

// App holds the RPC client used by CLI commands.
// The client is initialized lazily via the Before hook so that the
// --config flag value is available.
type App struct {
	client *client.Client
}

// connect initializes the RPC client from the config file. A missing config
// at the default path is not an error — [LoadConfig] falls back to validated
// in-memory defaults — but a missing config at an explicitly-passed --config
// path is, so the explicit-ness of the flag (not just its string value, which
// always carries the default) is passed through.
func (a *App) connect(cmd *cli.Command) error {
	if a.client != nil {
		return nil
	}

	configPath := ""
	if cmd.IsSet("config") {
		configPath = cmd.String("config")
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	a.client = client.New(cfg.SocketPath())

	return nil
}

// commonFlags returns the global flags shared by [NewApp] and
// [NewEmbeddedCommand].
func commonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			Value:   "~/.xdb/config.json",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output format (json, table, yaml, ndjson)",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Enable verbose logging",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
		},
	}
}

// newBaseCommand returns a *cli.Command pre-populated with the flags, output
// writers, and connect-on-Before wiring shared by [NewApp] and
// [NewEmbeddedCommand]. Callers set Name, Usage, Commands, and Action, then
// call [installUsageErrorHandler] once the tree is complete.
func (a *App) newBaseCommand(stdout, stderr io.Writer) *cli.Command {
	return &cli.Command{
		Flags:     commonFlags(),
		Writer:    stdout,
		ErrWriter: stderr,
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, a.connect(cmd)
		},
		ExitErrHandler: func(_ context.Context, cmd *cli.Command, err error) {
			// Render the error using the live --output flag from the command
			// that produced it, then let [main] set the exit code based on
			// [ExitCodeFor] after app.Run returns.
			WriteError(cmd.Root().ErrWriter, cmd.Root().String("output"), err)
		},
	}
}

// NewEmbeddedCommand creates an xdb CLI sub-command suitable for embedding
// inside another CLI (e.g. `lw db`). It omits lifecycle commands (daemon,
// init) that the host process owns, and uses name as the command name.
func NewEmbeddedCommand(name string) *cli.Command {
	a := &App{}

	root := a.newBaseCommand(os.Stdout, os.Stderr)
	root.Name = name
	root.Usage = "Query and manage xdb data"
	root.Commands = append(
		[]*cli.Command{
			a.recordsCmd(),
			a.schemasCmd(),
			a.namespacesCmd(),
			a.batchCmd(),
			a.importCmd(),
			a.exportCmd(),
			a.describeCmd(),
			skillsCmd(),
			contextCmd(),
		},
		a.aliasCommands()...,
	)
	root.Action = func(_ context.Context, cmd *cli.Command) error {
		_, err := fmt.Fprint(cmd.Root().Writer, agentContext)
		return err
	}

	installUsageErrorHandler(root)

	return root
}

// NewAppWithIO creates the root xdb CLI command, writing output and errors to
// stdout and stderr respectively. [NewApp] delegates to this with os.Stdout
// and os.Stderr; tests use it with in-memory buffers to capture output
// without touching the real terminal.
func NewAppWithIO(stdout, stderr io.Writer) *cli.Command {
	a := &App{}

	root := a.newBaseCommand(stdout, stderr)
	root.Name = "xdb"
	root.Usage = "An agent-first data layer. Model once, store anywhere."
	root.CustomRootCommandHelpTemplate = rootHelpTemplate
	root.Before = func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
		// cmd here is always the root command itself: urfave v3 runs each
		// ancestor's own Before hook against its own struct, and only the
		// root defines one. root.Args() (set once during the root's own
		// flag parse, and never mutated by child recursion) still holds the
		// first positional argument the user actually typed, so we use it
		// to skip connecting for commands that don't need a live client.
		switch cmd.Args().First() {
		case "init", "daemon", "context", "skills", "help", "":
			return ctx, nil
		default:
			return ctx, a.connect(cmd)
		}
	}
	root.Commands = append(
		[]*cli.Command{
			a.recordsCmd(),
			a.schemasCmd(),
			a.namespacesCmd(),
			a.batchCmd(),
			a.watchCmd(),
			a.importCmd(),
			a.exportCmd(),
			initCmd(),
			a.describeCmd(),
			skillsCmd(),
			contextCmd(),
			daemonCmd(),
		},
		a.aliasCommands()...,
	)
	root.Action = rootDispatch

	installUsageErrorHandler(root)

	return root
}

// NewApp creates the root xdb CLI command, writing to os.Stdout and os.Stderr.
func NewApp() *cli.Command {
	return NewAppWithIO(os.Stdout, os.Stderr)
}

// rootDispatch is the root command's Action. With no arguments, it shows the
// root help (exit 0). With an unrecognized first argument, it returns an
// INVALID_ARGUMENT envelope (exit 3) naming the command, prefixing a "did you
// mean" hint when urfave finds a plausible match among the registered
// commands.
func rootDispatch(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.ShowAppHelp(cmd)
	}

	name := cmd.Args().First()

	hint := "run 'xdb --help' for commands or 'xdb context' for the agent guide"
	if suggestion := cli.SuggestCommand(cmd.Commands, name); suggestion != "" {
		hint = fmt.Sprintf("did you mean 'xdb %s'? ", suggestion) + hint
	}

	return &output.ErrorEnvelope{
		Code:     CodeInvalidArgument,
		Message:  fmt.Sprintf("unknown command: %q", name),
		Resource: "cli",
		Action:   "dispatch",
		Hint:     hint,
	}
}

// --- Helpers ---

// readPayload reads a JSON payload from --json, --file, an explicit `-` positional
// token, or a piped stdin. An explicit `-` always reads stdin regardless of TTY.
func readPayload(cmd *cli.Command) (json.RawMessage, error) {
	if err := validateStdinInputs(cmd); err != nil {
		return nil, err
	}

	jsonFlag := cmd.String("json")
	fileFlag := cmd.String("file")

	hasJSON := jsonFlag != ""
	hasFile := fileFlag != "" && fileFlag != "-"
	hasDashFile := fileFlag == "-"
	hasDashArg := hasDashPositional(cmd)
	hasPipedStdin := !hasJSON && !hasFile && !hasDashFile && !hasDashArg && !isTerminal(os.Stdin)
	readStdin := hasDashFile || hasDashArg || hasPipedStdin

	if err := validate.MutuallyExclusive(map[string]bool{
		"json":  hasJSON,
		"file":  hasFile,
		"stdin": readStdin,
	}); err != nil {
		return nil, err
	}

	switch {
	case hasJSON:
		return json.RawMessage(jsonFlag), nil
	case hasFile:
		path, pathErr := validate.FilePath(fileFlag)
		if pathErr != nil {
			return nil, pathErr
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read file: %w", readErr)
		}

		return json.RawMessage(data), nil
	case readStdin:
		data, readErr := io.ReadAll(os.Stdin)
		if readErr != nil {
			return nil, fmt.Errorf("read stdin: %w", readErr)
		}

		return json.RawMessage(data), nil
	default:
		return nil, nil
	}
}

// hasDashPositional returns true when any positional argument is the literal `-`.
// `-` is the explicit "read from stdin" token.
func hasDashPositional(cmd *cli.Command) bool {
	args := cmd.Args().Slice()
	for _, a := range args {
		if a == "-" {
			return true
		}
	}

	return false
}

// validateStdinInputs returns an error when more than one input channel would
// consume stdin. stdin can be read at most once per command; a double consumer
// would silently truncate one of the inputs.
//
// Counted consumers: `--uri -`, `--file -`, and each positional `-` argument.
func validateStdinInputs(cmd *cli.Command) error {
	return checkStdinConsumers(cmd.String("uri"), cmd.String("file"), cmd.Args().Slice())
}

// checkStdinConsumers is the primitive form of [validateStdinInputs] used by tests.
func checkStdinConsumers(uri, file string, args []string) error {
	dashes := 0

	if uri == "-" {
		dashes++
	}

	if file == "-" {
		dashes++
	}

	for _, a := range args {
		if a == "-" {
			dashes++
		}
	}

	if dashes > 1 {
		return fmt.Errorf("at most one input may use `-` (stdin); cannot read both URI and payload from stdin")
	}

	return nil
}

// formatOne writes a single value using the appropriate formatter.
func formatOne(cmd *cli.Command, v any) error {
	w := cmd.Root().Writer
	flag := cmd.String("output")
	f := output.New(output.Detect(flag, isTerminalWriter(w)))

	return f.FormatOne(w, v)
}

// formatList writes a list using the appropriate formatter.
func formatList(cmd *cli.Command, items []any) error {
	w := cmd.Root().Writer
	flag := cmd.String("output")
	f := output.New(output.Detect(flag, isTerminalWriter(w)))

	return f.FormatList(w, items)
}

// formatPage writes a paginated list result using the appropriate
// formatter: an {items, total, next_offset} envelope for structured
// formats, bare items for ndjson and table.
func formatPage(cmd *cli.Command, page output.Page) error {
	w := cmd.Root().Writer
	flag := cmd.String("output")
	f := output.New(output.Detect(flag, isTerminalWriter(w)))

	return f.FormatPage(w, page)
}

// isTerminal returns true if the file is a terminal.
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}

	return (stat.Mode() & os.ModeCharDevice) != 0
}

// formatRawJSON unmarshals a json.RawMessage to a map and writes it.
func formatRawJSON(cmd *cli.Command, raw json.RawMessage) error {
	m, err := unmarshalPreserving(raw)
	if err != nil {
		return err
	}

	return formatOne(cmd, m)
}

// unmarshalPreserving parses raw JSON into a map without float64
// coercion, so large integers render verbatim instead of rounding
// through float64.
func unmarshalPreserving(raw json.RawMessage) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}

	return m, nil
}

// getURI returns the URI from --uri flag, a positional argument, or stdin when
// `-` is given. `-` is the explicit "read from stdin" token.
func getURI(cmd *cli.Command) (string, error) {
	if err := validateStdinInputs(cmd); err != nil {
		return "", err
	}

	uri := cmd.String("uri")
	if uri == "-" {
		return readURIFromStdin()
	}

	if uri != "" {
		return uri, nil
	}

	args := cmd.Args().Slice()
	for _, a := range args {
		if a == "-" {
			return readURIFromStdin()
		}

		if a != "" {
			return a, nil
		}
	}

	return "", fmt.Errorf("URI required (--uri flag or positional argument)")
}

// readURIFromStdin reads a single-line URI from stdin. Trailing whitespace and
// newlines are stripped.
func readURIFromStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// parseFields splits a comma-separated fields string into a slice.
func parseFields(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	fields := make([]string, 0, len(parts))

	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			fields = append(fields, trimmed)
		}
	}

	return fields
}
