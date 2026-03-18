package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func schemasCmd() *cli.Command {
	return &cli.Command{
		Name:               "schemas",
		Usage:              "Define and manage schema definitions",
		Category:           "resources",
		CustomHelpTemplate: subcommandHelpTemplate,
		Commands: []*cli.Command{
			{
				Name:               "create",
				Usage:              "Create a new schema (idempotent)",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              schemaMutationFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("schemas create: not implemented")
				},
			},
			{
				Name:               "get",
				Usage:              "Retrieve a schema definition",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              schemaReadFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("schemas get: not implemented")
				},
			},
			{
				Name:               "list",
				Usage:              "List schemas in a namespace",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              schemaListFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("schemas list: not implemented")
				},
			},
			{
				Name:               "update",
				Usage:              "Update a schema (patch semantics)",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              schemaMutationFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("schemas update: not implemented")
				},
			},
			{
				Name:               "delete",
				Usage:              "Delete a schema (idempotent, requires --force)",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              schemaDeleteFlags(),
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("schemas delete: not implemented")
				},
			},
		},
	}
}

func schemaMutationFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Schema URI", Required: true},
		&cli.StringFlag{Name: "json", Usage: "Inline JSON payload"},
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to input file"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Validate without writing"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
	}
}

func schemaReadFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Schema URI", Required: true},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
	}
}

func schemaListFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Namespace URI"},
		&cli.IntFlag{Name: "limit", Usage: "Max items per page"},
		&cli.IntFlag{Name: "offset", Usage: "Page offset"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
	}
}

func schemaDeleteFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Schema URI", Required: true},
		&cli.BoolFlag{Name: "force", Usage: "Confirm deletion", Required: true},
		&cli.BoolFlag{Name: "cascade", Usage: "Delete schema and all records"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Validate without deleting"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
	}
}
