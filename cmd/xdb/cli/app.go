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

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/validate"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// App holds the store, services, and encoder used by CLI commands.
type App struct {
	records    *api.RecordService
	schemas    *api.SchemaService
	namespaces *api.NamespaceService
	batch      *api.BatchService
	encoder    *xdbjson.Encoder
}

func newApp() *App {
	s := xdbmemory.New()

	return &App{
		records:    api.NewRecordService(s),
		schemas:    api.NewSchemaService(s),
		namespaces: api.NewNamespaceService(s),
		batch:      api.NewBatchService(s),
		encoder:    xdbjson.NewDefaultEncoder(),
	}
}

// NewApp creates the root xdb CLI command.
func NewApp() *cli.Command {
	a := newApp()

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
				contextCmd(),
				skillsCmd(),
				daemonCmd(),
			},
			a.aliasCommands()...,
		),
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("run 'xdb --help' for usage")
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

// recordToMap converts a core.Record to a map for output formatting.
func (a *App) recordToMap(rec *core.Record) (map[string]any, error) {
	data, err := a.encoder.FromRecord(rec)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if jsonErr := json.Unmarshal(data, &m); jsonErr != nil {
		return nil, jsonErr
	}

	return m, nil
}

// formatRecord encodes a record and writes it.
func (a *App) formatRecord(cmd *cli.Command, rec *core.Record) error {
	m, err := a.recordToMap(rec)
	if err != nil {
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
