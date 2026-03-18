package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func daemonCmd() *cli.Command {
	return &cli.Command{
		Name:               "daemon",
		Usage:              "Manage the XDB daemon (start, stop, status, restart)",
		Category:           "system",
		CustomHelpTemplate: subcommandHelpTemplate,
		Commands: []*cli.Command{
			{
				Name:               "start",
				Usage:              "Start the daemon",
				CustomHelpTemplate: commandHelpTemplate,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "foreground", Usage: "Run in foreground"},
				},
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("daemon start: not implemented")
				},
			},
			{
				Name:               "stop",
				Usage:              "Stop the daemon",
				CustomHelpTemplate: commandHelpTemplate,
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("daemon stop: not implemented")
				},
			},
			{
				Name:               "status",
				Usage:              "Show daemon status",
				CustomHelpTemplate: commandHelpTemplate,
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("daemon status: not implemented")
				},
			},
			{
				Name:               "restart",
				Usage:              "Restart the daemon",
				CustomHelpTemplate: commandHelpTemplate,
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("daemon restart: not implemented")
				},
			},
		},
	}
}
