package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/phsym/console-slog"
	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/cmd/xdb/app"
)

var (
	// Build information (set via ldflags).
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	setupLogger(slog.LevelWarn) // Default to warn level

	cmd := buildCLI()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer cancel()

	if err := cmd.Run(ctx, os.Args); err != nil {
		slog.Error("[CLI] Command failed", "error", err)
		os.Exit(1)
	}
}

func setupLogger(level slog.Level) {
	logger := slog.New(
		console.NewHandler(os.Stderr, &console.HandlerOptions{
			Level:     level,
			AddSource: false, // Disable by default, enable with --debug
		}),
	)
	slog.SetDefault(logger)
}

func buildCLI() *cli.Command {
	cmd := &cli.Command{
		Name:        "xdb",
		Usage:       "A tuple-based database CLI for managing schemas and records",
		Description: "XDB is a personal data store that provides a simple interface for storing and querying structured data using tuples and records.",
		Version:     formatVersion(),

		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			if cmd.Bool("debug") {
				setupLogger(slog.LevelDebug)
			} else if cmd.Bool("verbose") {
				setupLogger(slog.LevelInfo)
			}
			return ctx, nil
		},

		// Global flags available to all commands
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output format: json, table, yaml (auto-detected by default)",
				Sources: cli.EnvVars("XDB_OUTPUT"),
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "path to config file (defaults to xdb.yaml or xdb.yml)",
				Sources: cli.EnvVars("XDB_CONFIG"),
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "enable verbose logging",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "enable debug logging with source locations",
			},
		},
	}

	cmd.Commands = []*cli.Command{
		buildMakeSchemaCommand(),
		buildGetCommand(),
		buildPutCommand(),
		buildListCommand(),
		buildRemoveCommand(),
		buildDaemonCommand(),
		buildServerCommand(),
	}

	return cmd
}

func formatVersion() string {
	return Version + " (commit: " + GitCommit + ", built: " + BuildDate + ")"
}

func buildMakeSchemaCommand() *cli.Command {
	return &cli.Command{
		Name:        "make-schema",
		Category:    "Schema Management",
		Description: "creates or updates a schema at the given URI",
		UsageText: "xdb make-schema <uri> [--schema <file>]\n\n" +
			"Examples:\n" +
			"  xdb make-schema xdb://com.example/users --schema users.json\n" +
			"  xdb make-schema xdb://com.example/posts -s posts.json",
		Aliases: []string{"ms"},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "uri",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "schema",
				Aliases: []string{"s"},
				Usage:   "path to schema definition file (JSON)",
			},
		},
		Action: app.MakeSchema,
	}
}

func buildGetCommand() *cli.Command {
	return &cli.Command{
		Name:        "get",
		Category:    "Data Operations",
		Description: "retrieves a resource by its URI",
		UsageText: "xdb get <uri>\n\n" +
			"Examples:\n" +
			"  xdb get xdb://com.example/users/123\n" +
			"  xdb get xdb://com.example/users/123#name\n" +
			"  xdb get xdb://com.example/users --output json",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "uri",
			},
		},
		Action: app.Get,
	}
}

func buildPutCommand() *cli.Command {
	return &cli.Command{
		Name:        "put",
		Category:    "Data Operations",
		Description: "creates or updates a record",
		UsageText: "xdb put <uri> [--file <path>] [--format json|yaml]\n\n" +
			"Examples:\n" +
			"  xdb put xdb://com.example/users/123 --file user.json\n" +
			"  echo '{\"name\":\"Alice\"}' | xdb put xdb://com.example/users/123\n" +
			"  xdb put xdb://com.example/users/123 -f user.yaml --format yaml",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "uri",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "path to file (reads from stdin if omitted)",
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "input format: json (default) or yaml",
				Value: "json",
			},
		},
		Action: app.Put,
	}
}

func buildListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Category:    "Data Operations",
		Description: "lists resources matching the URI pattern",
		UsageText: "xdb list <pattern> [--limit N] [--offset N]\n\n" +
			"Examples:\n" +
			"  xdb list xdb://com.example/users\n" +
			"  xdb ls xdb://com.example --limit 10\n" +
			"  xdb ls xdb://com.example --limit 10 --offset 20",
		Aliases: []string{"ls"},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "pattern",
			},
		},
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "limit",
				Usage: "maximum number of results to return",
				Value: 100,
			},
			&cli.IntFlag{
				Name:  "offset",
				Usage: "number of results to skip",
				Value: 0,
			},
		},
		Action: app.List,
	}
}

func buildRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:        "remove",
		Category:    "Data Operations",
		Description: "deletes a resource by its URI",
		UsageText: "xdb remove <uri> [--force]\n\n" +
			"Examples:\n" +
			"  xdb remove xdb://com.example/users/123\n" +
			"  xdb rm xdb://com.example/users/123 --force",
		Aliases: []string{"rm", "delete"},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "uri",
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "skip confirmation prompt",
			},
		},
		Action: app.Remove,
	}
}

func buildDaemonCommand() *cli.Command {
	return &cli.Command{
		Name:        "daemon",
		Category:    "Daemon",
		Description: "manage the XDB daemon process",
		UsageText: "xdb daemon <action>\n\n" +
			"Actions:\n" +
			"  start     Start the daemon in background\n" +
			"  stop      Stop the running daemon\n" +
			"  status    Show daemon status and health\n" +
			"  restart   Restart the daemon\n" +
			"  logs      View daemon logs",
		Commands: []*cli.Command{
			buildDaemonStartCommand(),
			buildDaemonStopCommand(),
			buildDaemonStatusCommand(),
			buildDaemonRestartCommand(),
			buildDaemonLogsCommand(),
		},
	}
}

func buildDaemonStartCommand() *cli.Command {
	return &cli.Command{
		Name:        "start",
		Description: "start the daemon in background",
		UsageText: "xdb daemon start\n\n" +
			"Examples:\n" +
			"  xdb daemon start",
		Action: app.DaemonStart,
	}
}

func buildDaemonStopCommand() *cli.Command {
	return &cli.Command{
		Name:        "stop",
		Description: "stop the running daemon",
		UsageText: "xdb daemon stop [--force]\n\n" +
			"Examples:\n" +
			"  xdb daemon stop\n" +
			"  xdb daemon stop --force",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "force",
				Usage: "force kill if graceful shutdown fails",
			},
		},
		Action: app.DaemonStop,
	}
}

func buildDaemonStatusCommand() *cli.Command {
	return &cli.Command{
		Name:        "status",
		Description: "show daemon status and health",
		UsageText: "xdb daemon status [--json]\n\n" +
			"Examples:\n" +
			"  xdb daemon status\n" +
			"  xdb daemon status --json",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "output status as JSON",
			},
		},
		Action: app.DaemonStatus,
	}
}

func buildDaemonRestartCommand() *cli.Command {
	return &cli.Command{
		Name:        "restart",
		Description: "restart the daemon",
		UsageText: "xdb daemon restart [--force]\n\n" +
			"Examples:\n" +
			"  xdb daemon restart\n" +
			"  xdb daemon restart --force",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "force",
				Usage: "force kill if graceful shutdown fails",
			},
		},
		Action: app.DaemonRestart,
	}
}

func buildDaemonLogsCommand() *cli.Command {
	return &cli.Command{
		Name:        "logs",
		Description: "view daemon logs",
		UsageText: "xdb daemon logs [-f] [-n <lines>]\n\n" +
			"Examples:\n" +
			"  xdb daemon logs\n" +
			"  xdb daemon logs -f\n" +
			"  xdb daemon logs -n 50",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "follow log output (like tail -f)",
			},
			&cli.IntFlag{
				Name:    "lines",
				Aliases: []string{"n"},
				Usage:   "number of lines to show",
				Value:   100,
			},
		},
		Action: app.DaemonLogs,
	}
}

func buildServerCommand() *cli.Command {
	return &cli.Command{
		Name:        "server",
		Category:    "Daemon",
		Description: "deprecated: use 'xdb daemon start' instead",
		Hidden:      true,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return fmt.Errorf("the 'server' command has been removed; use 'xdb daemon start' instead")
		},
	}
}
