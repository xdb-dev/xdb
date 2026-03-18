package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func recordsCmd() *cli.Command {
	return &cli.Command{
		Name:               "records",
		Usage:              "Create, read, update, delete records",
		Category:           "resources",
		CustomHelpTemplate: subcommandHelpTemplate,
		Commands: []*cli.Command{
			{
				Name:               "create",
				Usage:              "Create a new record (idempotent)",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordMutationFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("records create: not implemented")
				},
			},
			{
				Name:               "get",
				Usage:              "Retrieve a record by URI",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordReadFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("records get: not implemented")
				},
			},
			{
				Name:               "list",
				Usage:              "List records in a schema",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordListFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("records list: not implemented")
				},
			},
			{
				Name:               "update",
				Usage:              "Update a record (patch semantics)",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordMutationFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("records update: not implemented")
				},
			},
			{
				Name:               "upsert",
				Usage:              "Create or replace a record",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordMutationFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("records upsert: not implemented")
				},
			},
			{
				Name:               "delete",
				Usage:              "Delete a record (idempotent, requires --force)",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordDeleteFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("records delete: not implemented")
				},
			},
		},
	}
}

func recordMutationFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Record URI", Required: true},
		&cli.StringFlag{Name: "json", Usage: "Inline JSON payload"},
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to input file"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Validate without writing"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		&cli.BoolFlag{Name: "quiet", Usage: "Suppress output"},
	}
}

func recordReadFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Record URI", Required: true},
		&cli.StringFlag{Name: "fields", Usage: "Comma-separated field mask"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
	}
}

func recordListFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Schema URI", Required: true},
		&cli.StringFlag{Name: "filter", Usage: "Human-friendly filter expression"},
		&cli.StringFlag{Name: "query", Usage: "Structured JSON query"},
		&cli.StringFlag{Name: "fields", Usage: "Comma-separated field mask"},
		&cli.IntFlag{Name: "limit", Usage: "Max items per page"},
		&cli.IntFlag{Name: "offset", Usage: "Page offset"},
		&cli.BoolFlag{Name: "page-all", Usage: "Stream all pages"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
	}
}

func recordDeleteFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Record URI", Required: true},
		&cli.BoolFlag{Name: "force", Usage: "Confirm deletion", Required: true},
		&cli.BoolFlag{Name: "dry-run", Usage: "Validate without deleting"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		&cli.BoolFlag{Name: "quiet", Usage: "Suppress output"},
	}
}
