package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func initCmd() *cli.Command {
	return &cli.Command{
		Name:               "init",
		Usage:              "Initialize XDB (create ~/.xdb/, config, start daemon)",
		Category:           "system",
		CustomHelpTemplate: commandHelpTemplate,
		Action:             initAction,
	}
}

func initAction(_ context.Context, cmd *cli.Command) error {
	configPath := DefaultConfigPath()

	created, err := EnsureConfigAt(configPath)
	if err != nil {
		return err
	}

	if created {
		fmt.Fprintf(os.Stderr, "Created %s\n", configPath)
	} else {
		fmt.Fprintf(os.Stderr, "Config already exists: %s\n", configPath)
	}

	cfg, loadErr := LoadConfig(configPath)
	if loadErr != nil {
		return loadErr
	}

	return formatOne(cmd, map[string]string{
		"status": "initialized",
		"dir":    cfg.ExpandedDir(),
		"config": configPath,
	})
}
