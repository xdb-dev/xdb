package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func initCmd() *cli.Command {
	return &cli.Command{
		Name:               "init",
		Usage:              "Initialize XDB (create ~/.xdb/, config, start daemon)",
		Category:           "system",
		CustomHelpTemplate: commandHelpTemplate,
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("init: not implemented")
		},
	}
}
