package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/phsym/console-slog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"github.com/xdb-dev/xdb/cmd/xdb/app"
)

func main() {
	logger := slog.New(
		console.NewHandler(os.Stderr, &console.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		}),
	)
	slog.SetDefault(logger)

	cmd := &cli.Command{
		Name:        "XDB",
		Description: "Your Personal Data Store",
	}

	cmd.Commands = []*cli.Command{
		{
			Name:        "server",
			Description: "starts XDB server",
			Usage:       "server [command]",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "path to config file (defaults to xdb.yaml or xdb.yml in current directory)",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				cfg, err := app.LoadConfig(ctx, cmd.String("config"))
				if err != nil {
					return err
				}

				server, err := app.NewServer(cfg)
				if err != nil {
					return err
				}

				return server.Run(ctx)
			},
		},
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer cancel()

	if err := cmd.Run(ctx, os.Args); err != nil {
		log.Error().Err(err).Msg("exit with error")
		os.Exit(1)
	}
}
