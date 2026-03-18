package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
)

func (a *App) batchCmd() *cli.Command {
	return &cli.Command{
		Name:               "batch",
		Usage:              "Execute multiple operations atomically",
		Category:           "operations",
		CustomHelpTemplate: commandHelpTemplate,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "json", Usage: "Inline JSON array of operations"},
			&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to operations file"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Validate without executing"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: a.batchExecute,
	}
}

func (a *App) batchExecute(ctx context.Context, cmd *cli.Command) error {
	data, err := readPayload(cmd)
	if err != nil {
		return err
	}

	if data == nil {
		return fmt.Errorf("batch requires a payload (--json, --file, or stdin)")
	}

	resp, err := a.batch.Execute(ctx, &api.ExecuteBatchRequest{
		Operations: data,
		DryRun:     cmd.Bool("dry-run"),
	})
	if err != nil {
		return err
	}

	return formatOne(cmd, resp)
}
