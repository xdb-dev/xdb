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
		Usage:              "Initialize XDB and start the daemon",
		Category:           "system",
		CustomHelpTemplate: commandHelpTemplate,
		Before: func(ctx context.Context, _ *cli.Command) (context.Context, error) {
			// init creates the config file, so it doesn't need to connect to the daemon first
			return ctx, nil
		},
		Action: initAction,
	}
}

func initAction(_ context.Context, cmd *cli.Command) error {
	configPath := expandTilde(cmd.Root().String("config"))

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

	// Create the XDB data directory.
	if mkErr := os.MkdirAll(cfg.ExpandedDir(), 0o700); mkErr != nil {
		return fmt.Errorf("create xdb directory: %w", mkErr)
	}

	// Start the daemon (best-effort — don't fail init if spawn fails).
	daemonStatus := "started"
	if isDaemonRunning(cfg) {
		daemonStatus = "already running"
	} else if spawnErr := spawnDaemon(cfg, configPath); spawnErr != nil {
		daemonStatus = "not started"
		fmt.Fprintf(os.Stderr, "Warning: could not start daemon: %v\n", spawnErr)
	}

	return formatOne(cmd, map[string]string{
		"status": "initialized",
		"dir":    cfg.ExpandedDir(),
		"config": configPath,
		"daemon": daemonStatus,
	})
}
