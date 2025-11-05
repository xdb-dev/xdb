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
			Name:        "make-repo",
			Description: "creates a new repository",
			Usage:       "make-repo [name] [--schema <schema_path>]",
			Aliases:     []string{"mr"},
			Arguments: []cli.Argument{
				&cli.StringArg{
					Name: "name",
				},
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "schema",
					Aliases: []string{"s"},
					Usage:   "path to schema file",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				name := cmd.String("name")
				schema := cmd.String("schema")

				slog.Info("[XDB] Making repo", "name", name, "schema", schema)

				return nil //app.MakeRepo(ctx, name, schema)
			},
		},
		{
			Name:        "get",
			Description: "gets a resource by its URI",
			Usage:       "get [uri]",
			Arguments: []cli.Argument{
				&cli.StringArg{
					Name: "uri",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				uri := cmd.String("uri")

				slog.Info("[XDB] Getting resource", "uri", uri)

				return nil //app.GetResource(ctx, uri)
			},
		},
		{
			Name:        "put",
			Description: "puts a resource by its URI",
			Usage:       "put [uri] [--file <file_path>]",
			Arguments: []cli.Argument{
				&cli.StringArg{
					Name: "uri",
				},
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "file",
					Aliases: []string{"f"},
					Usage:   "path to file to put",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				uri := cmd.String("uri")
				file := cmd.String("file")

				slog.Info("[XDB] Putting resource", "uri", uri, "file", file)

				return nil //app.PutResource(ctx, uri, file)
			},
		},
		{
			Name:        "list",
			Description: "lists all resources matching the given URI pattern",
			Usage:       "list [pattern]",
			Arguments: []cli.Argument{
				&cli.StringArg{
					Name: "uri_pattern",
				},
			},
			Aliases: []string{"ls"},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				pattern := cmd.String("pattern")

				slog.Info("[XDB] Listing resources", "pattern", pattern)

				return nil //app.ListRepos(ctx, pattern)
			},
		},
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
