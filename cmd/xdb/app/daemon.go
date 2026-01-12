package app

import (
	"context"

	"github.com/urfave/cli/v3"
)

// DaemonStart starts the daemon in the background.
func DaemonStart(ctx context.Context, cmd *cli.Command) error {
	cfg, err := LoadConfig(cmd.String("config"))
	if err != nil {
		return err
	}

	daemon := NewDaemon(cfg)
	return daemon.Start(cmd.String("config"))
}

// DaemonStop stops the running daemon.
func DaemonStop(ctx context.Context, cmd *cli.Command) error {
	cfg, err := LoadConfig(cmd.String("config"))
	if err != nil {
		return err
	}

	force := cmd.Bool("force")
	daemon := NewDaemon(cfg)
	return daemon.Stop(force)
}

// DaemonStatus shows daemon status and health.
func DaemonStatus(ctx context.Context, cmd *cli.Command) error {
	cfg, err := LoadConfig(cmd.String("config"))
	if err != nil {
		return err
	}

	daemon := NewDaemon(cfg)
	info, err := daemon.GetStatus(ctx)
	if err != nil {
		return err
	}

	jsonOutput := cmd.Bool("json")
	if err := PrintDaemonStatus(info, jsonOutput); err != nil {
		return err
	}

	if exitCode := StatusExitCode(info); exitCode != 0 {
		return cli.Exit("", exitCode)
	}

	return nil
}

// DaemonRestart restarts the daemon.
func DaemonRestart(ctx context.Context, cmd *cli.Command) error {
	cfg, err := LoadConfig(cmd.String("config"))
	if err != nil {
		return err
	}

	force := cmd.Bool("force")
	daemon := NewDaemon(cfg)
	return daemon.Restart(force, cmd.String("config"))
}

// DaemonLogs shows daemon logs.
func DaemonLogs(ctx context.Context, cmd *cli.Command) error {
	cfg, err := LoadConfig(cmd.String("config"))
	if err != nil {
		return err
	}

	follow := cmd.Bool("follow")
	lines := cmd.Int("lines")

	daemon := NewDaemon(cfg)
	if follow {
		return daemon.FollowLogs(ctx, int(lines))
	}

	return daemon.TailLogs(int(lines))
}
