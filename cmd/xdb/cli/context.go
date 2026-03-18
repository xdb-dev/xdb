package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func contextCmd() *cli.Command {
	return &cli.Command{
		Name:               "context",
		Usage:              "Print full agent context document",
		Category:           "agent",
		CustomHelpTemplate: commandHelpTemplate,
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("context: not implemented")
		},
	}
}
