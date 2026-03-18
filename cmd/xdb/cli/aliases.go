package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// aliasCommands returns human-friendly alias commands.
// Each alias resolves to a canonical resource command based on URI depth.
func aliasCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:               "get",
			Usage:              "Get a resource by URI",
			Category:           "aliases",
			CustomHelpTemplate: commandHelpTemplate,
			ArgsUsage:          "<uri>",
			Action: func(_ context.Context, _ *cli.Command) error {
				return fmt.Errorf("get: not implemented")
			},
		},
		{
			Name:               "put",
			Usage:              "Upsert a record by URI",
			Category:           "aliases",
			CustomHelpTemplate: commandHelpTemplate,
			ArgsUsage:          "<uri>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "json", Usage: "Inline JSON payload"},
				&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to input file"},
			},
			Action: func(_ context.Context, _ *cli.Command) error {
				return fmt.Errorf("put: not implemented")
			},
		},
		{
			Name:               "ls",
			Usage:              "List resources by URI",
			Category:           "aliases",
			CustomHelpTemplate: commandHelpTemplate,
			ArgsUsage:          "[uri]",
			Flags: []cli.Flag{
				&cli.IntFlag{Name: "limit", Usage: "Max items per page"},
				&cli.IntFlag{Name: "offset", Usage: "Page offset"},
				&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
			},
			Action: func(_ context.Context, _ *cli.Command) error {
				return fmt.Errorf("ls: not implemented")
			},
		},
		{
			Name:               "rm",
			Usage:              "Delete a resource by URI",
			Category:           "aliases",
			CustomHelpTemplate: commandHelpTemplate,
			ArgsUsage:          "<uri>",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "force", Usage: "Confirm deletion", Required: true},
			},
			Action: func(_ context.Context, _ *cli.Command) error {
				return fmt.Errorf("rm: not implemented")
			},
		},
		{
			Name:               "make-schema",
			Usage:              "Create a schema",
			Category:           "aliases",
			CustomHelpTemplate: commandHelpTemplate,
			ArgsUsage:          "<uri>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "json", Usage: "Inline JSON payload"},
				&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to input file"},
			},
			Action: func(_ context.Context, _ *cli.Command) error {
				return fmt.Errorf("make-schema: not implemented")
			},
		},
	}
}
