// Package cli implements the xdb command-line interface.
package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// NewApp creates the root xdb CLI command.
func NewApp() *cli.Command {
	return &cli.Command{
		Name:                          "xdb",
		Usage:                         "Tuple-based data store",
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
				recordsCmd(),
				schemasCmd(),
				namespacesCmd(),
				batchCmd(),
				watchCmd(),
				importCmd(),
				exportCmd(),
				initCmd(),
				schemaInspectCmd(),
				contextCmd(),
				skillsCmd(),
				daemonCmd(),
			},
			aliasCommands()...,
		),
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("run 'xdb --help' for usage")
		},
	}
}
