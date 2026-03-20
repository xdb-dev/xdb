// Package cli implements the xdb command-line interface.
package cli

import (
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

// connect initializes the RPC client from the config file.
func (a *App) connect(cmd *cli.Command) error {
	if a.client != nil {
		return nil
	}

	cfg, err := LoadConfig(cmd.String("config"))
	if err != nil {
		return err
	}

	a.client = client.New(cfg.SocketPath())

	return nil
}

// NewApp creates the root xdb CLI command.
func NewApp() *cli.Command {
	a := &App{}

	return &cli.Command{
		Name:                          "xdb",
		Usage:                         "An agent-first data layer. Model once, store anywhere.",
		CustomRootCommandHelpTemplate: rootHelpTemplate,
		Flags: []cli.Flag{
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
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, a.connect(cmd)
		},
		Commands: append(
			[]*cli.Command{
				a.recordsCmd(),
				a.schemasCmd(),
				a.namespacesCmd(),
				a.batchCmd(),
				watchCmd(),
				a.importCmd(),
				a.exportCmd(),
				initCmd(),
				a.describeCmd(),
				skillsCmd(),
				daemonCmd(),
			},
			a.aliasCommands()...,
		),
		Action: func(_ context.Context, _ *cli.Command) error {
			_, err := fmt.Fprint(os.Stdout, agentContext)
			return err
		},
	}
}

// --- Helpers ---

// readPayload reads a JSON payload from --json, --file, or stdin.
func readPayload(cmd *cli.Command) (json.RawMessage, error) {
	jsonFlag := cmd.String("json")
	fileFlag := cmd.String("file")

	hasJSON := jsonFlag != ""
	hasFile := fileFlag != ""
	hasStdin := !hasJSON && !hasFile && !isTerminal(os.Stdin)

	if err := validate.MutuallyExclusive(map[string]bool{
		"json": hasJSON,
		"file": hasFile,
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
	case hasStdin:
		data, readErr := io.ReadAll(os.Stdin)
		if readErr != nil {
			return nil, fmt.Errorf("read stdin: %w", readErr)
		}

		return json.RawMessage(data), nil
	default:
		return nil, nil
	}
}

// formatOne writes a single value using the appropriate formatter.
func formatOne(cmd *cli.Command, v any) error {
	flag := cmd.String("output")
	isTTY := isTerminal(os.Stdout)
	f := output.New(output.Detect(flag, isTTY))

	return f.FormatOne(os.Stdout, v)
}

// formatList writes a list using the appropriate formatter.
func formatList(cmd *cli.Command, items []any) error {
	flag := cmd.String("output")
	isTTY := isTerminal(os.Stdout)
	f := output.New(output.Detect(flag, isTTY))

	return f.FormatList(os.Stdout, items)
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
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}

	return formatOne(cmd, m)
}

// getURI returns the URI from --uri flag or first positional argument.
func getURI(cmd *cli.Command) (string, error) {
	uri := cmd.String("uri")
	if uri != "" {
		return uri, nil
	}

	args := cmd.Args()
	if args.Len() > 0 {
		return args.First(), nil
	}

	return "", fmt.Errorf("URI required (--uri flag or positional argument)")
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
