package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/cmd/xdb/daemon"
)

func daemonCmd() *cli.Command {
	return &cli.Command{
		Name:               "daemon",
		Usage:              "Manage the XDB daemon (start, stop, status, restart)",
		Category:           "system",
		CustomHelpTemplate: subcommandHelpTemplate,
		Commands: []*cli.Command{
			{
				Name:               "start",
				Usage:              "Start the daemon",
				CustomHelpTemplate: commandHelpTemplate,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "foreground", Usage: "Run in foreground"},
				},
				Action: daemonStartAction,
			},
			{
				Name:               "stop",
				Usage:              "Stop the daemon",
				CustomHelpTemplate: commandHelpTemplate,
				Action:             daemonStopAction,
			},
			{
				Name:               "status",
				Usage:              "Show daemon status",
				CustomHelpTemplate: commandHelpTemplate,
				Action:             daemonStatusAction,
			},
			{
				Name:               "restart",
				Usage:              "Restart the daemon",
				CustomHelpTemplate: commandHelpTemplate,
				Action:             daemonRestartAction,
			},
		},
	}
}

func xdbDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".xdb")
	}

	return filepath.Join(home, ".xdb")
}

func daemonConfig() daemon.Config {
	dir := xdbDir()

	return daemon.Config{
		SocketPath: filepath.Join(dir, "xdb.sock"),
		LogFile:    filepath.Join(dir, "xdb.log"),
		Version:    "dev",
	}
}

func daemonStartAction(ctx context.Context, cmd *cli.Command) error {
	cfg := daemonConfig()

	// Ensure directory exists.
	if mkErr := os.MkdirAll(filepath.Dir(cfg.SocketPath), 0o700); mkErr != nil {
		return fmt.Errorf("create xdb directory: %w", mkErr)
	}

	d := daemon.New(cfg)

	fmt.Fprintf(os.Stderr, "Starting daemon on %s\n", cfg.SocketPath)

	return d.Start(ctx)
}

func daemonStopAction(_ context.Context, cmd *cli.Command) error {
	cfg := daemonConfig()
	d := daemon.New(cfg)

	status, err := d.Status()
	if err != nil {
		return err
	}

	if status == "stopped" {
		fmt.Fprintln(os.Stderr, "Daemon is not running")
		return nil
	}

	if stopErr := d.Stop(); stopErr != nil {
		return stopErr
	}

	fmt.Fprintln(os.Stderr, "Daemon stopped")

	return nil
}

func daemonStatusAction(_ context.Context, cmd *cli.Command) error {
	cfg := daemonConfig()
	d := daemon.New(cfg)

	status, err := d.Status()
	if err != nil {
		return err
	}

	return formatOne(cmd, map[string]string{
		"status": status,
		"socket": cfg.SocketPath,
	})
}

func daemonRestartAction(ctx context.Context, cmd *cli.Command) error {
	cfg := daemonConfig()
	d := daemon.New(cfg)

	// Stop if running.
	status, _ := d.Status()
	if status == "running" {
		if stopErr := d.Stop(); stopErr != nil {
			return stopErr
		}

		fmt.Fprintln(os.Stderr, "Daemon stopped")
	}

	return daemonStartAction(ctx, cmd)
}
