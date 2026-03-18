package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func batchCmd() *cli.Command {
	return &cli.Command{
		Name:               "batch",
		Usage:              "Execute multiple operations atomically",
		Category:           "operations",
		CustomHelpTemplate: commandHelpTemplate,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "json", Usage: "Inline JSON array of operations"},
			&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to operations file"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Validate without executing"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("batch: not implemented")
		},
	}
}
