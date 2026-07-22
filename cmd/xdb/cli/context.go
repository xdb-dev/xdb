package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// contextCmd prints the embedded agent-oriented CLI guide (CONTEXT.md).
// It replaces the old behavior of dumping the guide for any unrecognized
// invocation — see the root Action in app.go.
func contextCmd() *cli.Command {
	return &cli.Command{
		Name:               "context",
		Usage:              "Print the agent-oriented CLI guide",
		Category:           "agent",
		CustomHelpTemplate: commandHelpTemplate,
		Action: func(_ context.Context, cmd *cli.Command) error {
			_, err := fmt.Fprint(cmd.Root().Writer, agentContext)
			return err
		},
	}
}
