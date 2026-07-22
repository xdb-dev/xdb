package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
)

// installUsageErrorHandler sets OnUsageError on cmd and every command in its
// subtree, so that flag-parse and positional-argument-parse failures render
// as an INVALID_ARGUMENT envelope instead of urfave's bare "Incorrect Usage"
// text. Must be called after the full command tree — including aliases — is
// built.
func installUsageErrorHandler(cmd *cli.Command) {
	cmd.OnUsageError = usageErrorEnvelope

	for _, sub := range cmd.Commands {
		installUsageErrorHandler(sub)
	}
}

// usageErrorEnvelope builds the INVALID_ARGUMENT envelope installed by
// [installUsageErrorHandler]. Resource is the offending command's parent
// name, or "cli" at the root; Action is the offending command's own name.
func usageErrorEnvelope(_ context.Context, cmd *cli.Command, err error, _ bool) error {
	lineage := cmd.Lineage()

	resource := "cli"
	if len(lineage) > 1 {
		resource = lineage[1].Name
	}

	return &output.ErrorEnvelope{
		Code:     CodeInvalidArgument,
		Message:  err.Error(),
		Resource: resource,
		Action:   cmd.Name,
		Hint:     fmt.Sprintf("run '%s --help'", cmd.FullName()),
	}
}
