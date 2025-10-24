package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/xdb-dev/xdb/cmd/xdb/server"
)

func main() {
	cliApp := cli.NewApp()
	cliApp.Name = "XDB"
	cliApp.Description = "Your Personal Data Store"

	cliApp.Commands = []*cli.Command{
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
			Action: func(ctx *cli.Context) error {
				cfg, err := server.LoadConfig(ctx.Context, ctx.String("config"))
				if err != nil {
					return err
				}

				server := server.New(cfg)

				return server.Run(ctx.Context)
			},
		},
	}

	if err := cliApp.Run(os.Args); err != nil {
		fmt.Printf("exit with error: %+v\n", err)
		log.Error().Err(err).Msg("exit with error")
		panic(err)
	}
}
