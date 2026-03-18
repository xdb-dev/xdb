package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func skillsCmd() *cli.Command {
	return &cli.Command{
		Name:               "skills",
		Usage:              "Discover and fetch skill documents",
		Category:           "agent",
		CustomHelpTemplate: subcommandHelpTemplate,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Commands: []*cli.Command{
			{
				Name:               "get",
				Usage:              "Fetch a specific skill",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "<skill-name>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
				},
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("skills get: not implemented")
				},
			},
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("skills: not implemented")
		},
	}
}
