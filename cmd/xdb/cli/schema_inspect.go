package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func schemaInspectCmd() *cli.Command {
	return &cli.Command{
		Name:               "schema",
		Usage:              "Introspect methods, types, and data schemas",
		Category:           "agent",
		CustomHelpTemplate: commandHelpTemplate,
		ArgsUsage:          "[method.name | TypeName]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "methods", Usage: "List all methods"},
			&cli.BoolFlag{Name: "types", Usage: "List all types"},
			&cli.BoolFlag{Name: "value-types", Usage: "List supported value types"},
			&cli.StringFlag{Name: "uri", Usage: "Data schema URI"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("schema: not implemented")
		},
	}
}
