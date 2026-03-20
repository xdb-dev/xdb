package cli

import (
	"context"
	"encoding/json"
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

	var resp json.RawMessage
	if err := a.client.Call(ctx, "batch.execute", &api.ExecuteBatchRequest{
		Operations: data,
		DryRun:     cmd.Bool("dry-run"),
	}, &resp); err != nil {
		return err
	}

	var m map[string]any
	if jsonErr := json.Unmarshal(resp, &m); jsonErr != nil {
		return jsonErr
	}

	return formatOne(cmd, m)
}
