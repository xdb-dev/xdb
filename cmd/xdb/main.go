package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
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
			Action: func(ctx *cli.Context) error {
				cfg, err := LoadConfig(ctx.Context)
				if err != nil {
					return err
				}

				server := NewServer(cfg)

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
