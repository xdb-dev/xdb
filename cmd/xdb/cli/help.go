package cli

import (
	urfave "github.com/urfave/cli/v3"
)

func init() {
	// Override the global SubcommandHelpTemplate so that resource-level
	// commands (records, schemas, etc.) that set CustomHelpTemplate
	// can use it. The default ShowSubcommandHelp hardcodes the global
	// template, so we replace it with our subcommandHelpTemplate.
	urfave.SubcommandHelpTemplate = subcommandHelpTemplate
}
