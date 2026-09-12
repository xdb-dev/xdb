package cli

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
	"github.com/xdb-dev/xdb/schema"
)

// dryRunIgnoredError reports a daemon that accepted a dry_run request
// but returned no dry-run marker — an older daemon that executed the
// write for real. Surfaced as INTERNAL so agents treat it as a fault,
// not a validation result.
func dryRunIgnoredError(resource, action, uri string) error {
	return &output.ErrorEnvelope{
		Code:     CodeInternal,
		Message:  "daemon ignored dry_run — upgrade the daemon",
		Resource: resource,
		Action:   action,
		URI:      uri,
	}
}

// formatDryRun renders a validate-only response: the dry-run verdict
// plus the canonicalized record or definition that would be written.
// data holds a [json.RawMessage] for a record, a [schema.Def] for a
// definition, and nil for a delete.
//
// A table shows the verdict, a blank line, and then the record as `get`
// shows it. The structured formats carry the record under "record".
func formatDryRun(cmd *cli.Command, result *api.DryRunResult, data any) error {
	verdict := map[string]any{
		"dry_run": true,
		"valid":   result.Valid,
		"would":   result.Would,
	}

	data = dryRunData(data)

	format, err := output.Detect(cmd.String("output"), isTerminalWriter(cmd.Root().Writer))
	if err != nil {
		return err
	}

	if format != output.FormatTable {
		if data != nil {
			verdict["record"] = data
		}

		return formatOne(cmd, verdict)
	}

	if err := formatOne(cmd, verdict); err != nil {
		return err
	}

	if data == nil {
		return nil
	}

	if _, err := fmt.Fprintln(cmd.Root().Writer); err != nil {
		return err
	}

	if raw, ok := data.(json.RawMessage); ok {
		return formatRawJSON(cmd, raw)
	}

	return formatOne(cmd, data)
}

// dryRunData returns nil for an absent record, so that an empty raw
// message or a nil definition never reaches the output as "record".
func dryRunData(data any) any {
	switch d := data.(type) {
	case json.RawMessage:
		if len(d) == 0 {
			return nil
		}
	case *schema.Def:
		if d == nil {
			return nil
		}
	}

	return data
}
