package cli

import (
	"bytes"
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
			&cli.StringFlag{Name: "json", Usage: "Inline JSON (array or NDJSON)"},
			&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to operations file"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Validate without executing"},
			&cli.BoolFlag{Name: "non-atomic", Usage: "Allow sequential best-effort execution on non-transactional backends"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: a.batchExecute,
	}
}

func (a *App) batchExecute(ctx context.Context, cmd *cli.Command) error {
	data, err := readPayload(cmd)
	if err != nil {
		return invalidArgError("batch", "execute", err)
	}

	if data == nil {
		return invalidArgError("batch", "execute", fmt.Errorf("batch requires a payload (--json, --file, or stdin)"))
	}

	ops, err := normalizeBatchOps(data)
	if err != nil {
		return invalidArgError("batch", "execute", err)
	}

	var resp api.ExecuteBatchResponse
	if err := a.client.Call(ctx, "batch.execute", &api.ExecuteBatchRequest{
		Operations: ops,
		DryRun:     cmd.Bool("dry-run"),
		NonAtomic:  cmd.Bool("non-atomic"),
	}, &resp); err != nil {
		return wrapRPCError("batch", "execute", "", err)
	}

	return formatOne(cmd, resp)
}

// normalizeBatchOps parses the payload into typed operations. Accepts:
//
//   - JSON array:    [{"op":"records.create","uri":"...","data":{...}}, ...]
//   - NDJSON stream: one operation object per line
//
// The shape is detected by the first non-whitespace byte (`[` = array,
// `{` = ndjson).
func normalizeBatchOps(data json.RawMessage) ([]api.BatchOperation, error) {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty batch payload")
	}

	if trimmed[0] == '[' {
		var ops []api.BatchOperation
		if err := json.Unmarshal(trimmed, &ops); err != nil {
			return nil, fmt.Errorf("parse batch operations: %w", err)
		}
		return ops, nil
	}

	if trimmed[0] != '{' {
		return nil, fmt.Errorf("batch payload must be a JSON array or NDJSON stream of operation objects")
	}

	ops := make([]api.BatchOperation, 0)
	dec := json.NewDecoder(bytes.NewReader(trimmed))

	for dec.More() {
		var o api.BatchOperation
		if err := dec.Decode(&o); err != nil {
			return nil, fmt.Errorf("parse ndjson operation %d: %w", len(ops)+1, err)
		}

		ops = append(ops, o)
	}

	return ops, nil
}
