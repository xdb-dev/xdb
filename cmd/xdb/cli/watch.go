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
		Action: watchAction,
	}
}

func watchAction(_ context.Context, _ *cli.Command) error {
	// Watch requires pubsub infrastructure in the daemon.
	// This will be implemented when the daemon supports change streams.
	return fmt.Errorf("watch requires a running daemon with pubsub support (not yet implemented)")
}
