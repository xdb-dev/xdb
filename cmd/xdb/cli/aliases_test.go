package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// rm requires --force, the same as delete. The help must not read as if
// the alias passes --force for the caller.
func TestAliasRm_HelpSaysForceIsRequired(t *testing.T) {
	app := &App{}

	var rm *cli.Command
	for _, cmd := range app.aliasCommands() {
		if cmd.Name == "rm" {
			rm = cmd
		}
	}
	require.NotNil(t, rm)

	assert.Contains(t, rm.Description, "--force is required")
	assert.NotContains(t, rm.Description, "delete <uri> --force`")
}
