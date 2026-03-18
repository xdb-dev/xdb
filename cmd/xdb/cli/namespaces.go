package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func namespacesCmd() *cli.Command {
	return &cli.Command{
		Name:               "namespaces",
		Usage:              "List and inspect namespaces",
		Category:           "resources",
		CustomHelpTemplate: subcommandHelpTemplate,
		Commands: []*cli.Command{
			{
				Name:               "list",
				Usage:              "List all namespaces",
				CustomHelpTemplate: commandHelpTemplate,
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "limit", Usage: "Max items per page"},
					&cli.IntFlag{Name: "offset", Usage: "Page offset"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
				},
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("namespaces list: not implemented")
				},
			},
			{
				Name:               "get",
				Usage:              "Get namespace details",
				CustomHelpTemplate: commandHelpTemplate,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "uri", Usage: "Namespace URI", Required: true},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
				},
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("namespaces get: not implemented")
				},
			},
		},
	}
}
