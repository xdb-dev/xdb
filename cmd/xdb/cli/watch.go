package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func watchCmd() *cli.Command {
	return &cli.Command{
		Name:               "watch",
		Usage:              "Stream change notifications as NDJSON",
		Category:           "operations",
		CustomHelpTemplate: commandHelpTemplate,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "uri", Usage: "URI to watch", Required: true},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("watch: not implemented")
		},
	}
}
