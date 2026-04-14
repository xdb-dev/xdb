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

	var resp json.RawMessage
	if err := a.client.Call(ctx, "batch.execute", &api.ExecuteBatchRequest{
		Operations: ops,
		DryRun:     cmd.Bool("dry-run"),
	}, &resp); err != nil {
		return wrapRPCError("batch", "execute", "", err)
	}

	var m map[string]any
	if jsonErr := json.Unmarshal(resp, &m); jsonErr != nil {
		return jsonErr
	}

	return formatOne(cmd, m)
}

// normalizeBatchOps returns a JSON array of operations. Accepts two forms:
//
//   - JSON array:       [{"resource":"records","action":"create",...}, ...]
//   - NDJSON stream:    one operation object per line
//
// The shape is detected by the first non-whitespace byte (`[` = array, `{` = ndjson).
// Each operation mirrors the CLI grammar: `{resource, action, uri, payload}`.
func normalizeBatchOps(data json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty batch payload")
	}

	if trimmed[0] == '[' {
		return data, nil
	}

	if trimmed[0] != '{' {
		return nil, fmt.Errorf("batch payload must be a JSON array or NDJSON stream of operation objects")
	}

	ops := make([]json.RawMessage, 0)
	dec := json.NewDecoder(bytes.NewReader(trimmed))

	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("parse ndjson operation %d: %w", len(ops)+1, err)
		}

		ops = append(ops, raw)
	}

	encoded, err := json.Marshal(ops)
	if err != nil {
		return nil, fmt.Errorf("marshal normalized ops: %w", err)
	}

	return encoded, nil
}
