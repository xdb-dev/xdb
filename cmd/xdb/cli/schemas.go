package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
)

func (a *App) schemasCmd() *cli.Command {
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
				ArgsUsage:          "[URI]",
				Flags:              schemaMutationFlags(),
				Action:             a.schemaCreate,
			},
			{
				Name:               "get",
				Usage:              "Retrieve a schema definition",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "[URI]",
				Flags:              schemaReadFlags(),
				Action:             a.schemaGet,
			},
			{
				Name:               "list",
				Usage:              "List schemas in a namespace",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "[NAMESPACE_URI]",
				Flags:              schemaListFlags(),
				Action:             a.schemaList,
			},
			{
				Name:               "update",
				Usage:              "Update a schema (patch semantics)",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "[URI]",
				Flags:              schemaMutationFlags(),
				Action:             a.schemaUpdate,
			},
			{
				Name:               "delete",
				Usage:              "Delete a schema (idempotent, requires --force)",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "[URI]",
				Flags:              schemaDeleteFlags(),
				Action:             a.schemaDelete,
			},
		},
	}
}

func (a *App) schemaCreate(ctx context.Context, cmd *cli.Command) error {
	data, err := readPayload(cmd)
	if err != nil {
		return invalidArgError("schemas", "create", err)
	}

	if data == nil {
		return invalidArgError("schemas", "create", fmt.Errorf("create requires a payload (--json, --file, or stdin)"))
	}

	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("schemas", "create", err)
	}

	var resp api.CreateSchemaResponse
	if err := a.client.Call(ctx, "schemas.create", &api.CreateSchemaRequest{
		URI:  uri,
		Data: data,
	}, &resp); err != nil {
		return wrapRPCError("schemas", "create", uri, err)
	}

	return formatOne(cmd, resp.Data)
}

func (a *App) schemaGet(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("schemas", "get", err)
	}

	var resp api.GetSchemaResponse
	if err := a.client.Call(ctx, "schemas.get", &api.GetSchemaRequest{
		URI: uri,
	}, &resp); err != nil {
		return wrapRPCError("schemas", "get", uri, err)
	}

	return formatOne(cmd, resp.Data)
}

func (a *App) schemaList(ctx context.Context, cmd *cli.Command) error {
	// URI is optional for schemas list — uses flag or positional arg.
	uri, _ := getURI(cmd)

	var resp api.ListSchemasResponse
	if err := a.client.Call(ctx, "schemas.list", &api.ListSchemasRequest{
		URI:    uri,
		Limit:  int(cmd.Int("limit")),
		Offset: int(cmd.Int("offset")),
	}, &resp); err != nil {
		return wrapRPCError("schemas", "list", uri, err)
	}

	items := make([]any, len(resp.Items))
	for i, def := range resp.Items {
		items[i] = def
	}

	return formatList(cmd, items)
}

func (a *App) schemaUpdate(ctx context.Context, cmd *cli.Command) error {
	data, err := readPayload(cmd)
	if err != nil {
		return invalidArgError("schemas", "update", err)
	}

	if data == nil {
		return invalidArgError("schemas", "update", fmt.Errorf("update requires a payload (--json, --file, or stdin)"))
	}

	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("schemas", "update", err)
	}

	var resp api.UpdateSchemaResponse
	if err := a.client.Call(ctx, "schemas.update", &api.UpdateSchemaRequest{
		URI:  uri,
		Data: data,
	}, &resp); err != nil {
		return wrapRPCError("schemas", "update", uri, err)
	}

	return formatOne(cmd, resp.Data)
}

func (a *App) schemaDelete(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("schemas", "delete", err)
	}

	if err := a.client.Call(ctx, "schemas.delete", &api.DeleteSchemaRequest{
		URI:     uri,
		Cascade: cmd.Bool("cascade"),
	}, nil); err != nil {
		return wrapRPCError("schemas", "delete", uri, err)
	}

	return formatOne(cmd, map[string]string{
		"status": "deleted",
		"uri":    uri,
	})
}

func schemaMutationFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Schema URI"},
		&cli.StringFlag{Name: "json", Usage: "Inline JSON payload"},
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to input file"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Validate without writing"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
	}
}

func schemaReadFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Schema URI"},
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
		&cli.StringFlag{Name: "uri", Usage: "Schema URI"},
		&cli.BoolFlag{Name: "force", Usage: "Confirm deletion"},
		&cli.BoolFlag{Name: "cascade", Usage: "Delete schema and all records"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Validate without deleting"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
	}
}
