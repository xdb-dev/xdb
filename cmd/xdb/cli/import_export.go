package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func importCmd() *cli.Command {
	return &cli.Command{
		Name:               "import",
		Usage:              "Import records from NDJSON",
		Category:           "operations",
		CustomHelpTemplate: commandHelpTemplate,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "uri", Usage: "Target schema URI", Required: true},
			&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to NDJSON file"},
			&cli.BoolFlag{Name: "create-only", Usage: "Use create instead of upsert"},
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("import: not implemented")
		},
	}
}

func exportCmd() *cli.Command {
	return &cli.Command{
		Name:               "export",
		Usage:              "Export records as NDJSON",
		Category:           "operations",
		CustomHelpTemplate: commandHelpTemplate,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "uri", Usage: "Schema URI", Required: true},
			&cli.StringFlag{Name: "fields", Usage: "Comma-separated field mask"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("export: not implemented")
		},
	}
}
