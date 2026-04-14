package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/core"
)

func (a *App) aliasCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:               "get",
			Usage:              "Shorthand for <resource> get (records/schemas/namespaces by URI depth)",
			Description:        "Dispatches to records/schemas/namespaces get based on URI depth.\n   Equivalent to `xdb <resource> get <uri>`.",
			Category:           "aliases",
			CustomHelpTemplate: commandHelpTemplate,
			ArgsUsage:          "<uri>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "fields", Usage: "Comma-separated field mask"},
				&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
			},
			Action: a.aliasGet,
		},
		{
			Name:               "put",
			Usage:              "Shorthand for records upsert",
			Description:        "Equivalent to `xdb records upsert <uri> [--json|--file|-]`.",
			Category:           "aliases",
			CustomHelpTemplate: commandHelpTemplate,
			ArgsUsage:          "<uri>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "json", Usage: "Inline JSON payload"},
				&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to input file"},
				&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
			},
			Action: a.aliasPut,
		},
		{
			Name:               "ls",
			Usage:              "Shorthand for <resource> list (records/schemas/namespaces by URI depth)",
			Description:        "Dispatches to records/schemas/namespaces list based on URI depth.\n   No URI lists namespaces. Equivalent to `xdb <resource> list [uri]`.",
			Category:           "aliases",
			CustomHelpTemplate: commandHelpTemplate,
			ArgsUsage:          "[uri]",
			Flags: []cli.Flag{
				&cli.IntFlag{Name: "limit", Usage: "Max items per page"},
				&cli.IntFlag{Name: "offset", Usage: "Page offset"},
				&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
			},
			Action: a.aliasLs,
		},
		{
			Name:               "rm",
			Usage:              "Shorthand for <resource> delete (records or schemas by URI depth)",
			Description:        "Dispatches to records/schemas delete based on URI depth.\n   Equivalent to `xdb <resource> delete <uri> --force`.",
			Category:           "aliases",
			CustomHelpTemplate: commandHelpTemplate,
			ArgsUsage:          "<uri>",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "force", Usage: "Confirm deletion", Required: true},
				&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
			},
			Action: a.aliasRm,
		},
		{
			Name:               "make-schema",
			Usage:              "Shorthand for schemas create",
			Description:        "Equivalent to `xdb schemas create <uri> [--json|--file|-]`.",
			Category:           "aliases",
			CustomHelpTemplate: commandHelpTemplate,
			ArgsUsage:          "<uri>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "json", Usage: "Inline JSON payload"},
				&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to input file"},
				&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
			},
			Action: a.aliasMakeSchema,
		},
	}
}

// uriDepth returns the number of path components in an XDB URI.
func uriDepth(raw string) (int, error) {
	uri, err := core.ParseURI(raw)
	if err != nil {
		return 0, err
	}

	depth := 1
	if uri.Schema() != nil {
		depth++
	}

	if uri.ID() != nil {
		depth++
	}

	return depth, nil
}

func (a *App) aliasGet(ctx context.Context, cmd *cli.Command) error {
	depth, err := uriDepth(cmd.Args().First())
	if err != nil {
		return err
	}

	switch depth {
	case 3:
		return a.recordGet(ctx, cmd)
	case 2:
		return a.schemaGet(ctx, cmd)
	case 1:
		return a.namespaceGet(ctx, cmd)
	default:
		return fmt.Errorf("cannot infer resource from URI: %s", cmd.Args().First())
	}
}

func (a *App) aliasPut(ctx context.Context, cmd *cli.Command) error {
	return a.recordUpsert(ctx, cmd)
}

func (a *App) aliasLs(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return a.namespaceList(ctx, cmd)
	}

	depth, err := uriDepth(cmd.Args().First())
	if err != nil {
		return err
	}

	switch depth {
	case 2:
		return a.recordList(ctx, cmd)
	case 1:
		return a.schemaList(ctx, cmd)
	default:
		return fmt.Errorf("cannot infer list target from URI: %s", cmd.Args().First())
	}
}

func (a *App) aliasRm(ctx context.Context, cmd *cli.Command) error {
	depth, err := uriDepth(cmd.Args().First())
	if err != nil {
		return err
	}

	switch depth {
	case 3:
		return a.recordDelete(ctx, cmd)
	case 2:
		return a.schemaDelete(ctx, cmd)
	default:
		return fmt.Errorf("cannot infer resource to delete from URI: %s", cmd.Args().First())
	}
}

func (a *App) aliasMakeSchema(ctx context.Context, cmd *cli.Command) error {
	return a.schemaCreate(ctx, cmd)
}
