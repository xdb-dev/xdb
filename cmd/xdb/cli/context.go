package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	_ "embed"
)

//go:embed CONTEXT.md
var agentContext string

func contextCmd() *cli.Command {
	return &cli.Command{
		Name:               "context",
		Usage:              "Print full agent context document",
		Category:           "agent",
		CustomHelpTemplate: commandHelpTemplate,
		Action: func(_ context.Context, _ *cli.Command) error {
			_, err := fmt.Fprint(os.Stdout, agentContext)
			return err
		},
	}
}
