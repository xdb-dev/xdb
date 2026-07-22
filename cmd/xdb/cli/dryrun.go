package cli

import (
	"encoding/json"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
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
func formatDryRun(cmd *cli.Command, result *api.DryRunResult, data json.RawMessage) error {
	doc := map[string]any{
		"dry_run": true,
		"valid":   result.Valid,
		"would":   result.Would,
	}
	if len(data) > 0 {
		doc["record"] = json.RawMessage(data)
	}

	return formatOne(cmd, doc)
}
