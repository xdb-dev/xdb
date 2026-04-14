package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
)

func (a *App) recordsCmd() *cli.Command {
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
				Action:             a.recordCreate,
			},
			{
				Name:               "get",
				Usage:              "Retrieve a record by URI",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordReadFlags(),
				Action:             a.recordGet,
			},
			{
				Name:               "list",
				Usage:              "List records in a schema",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordListFlags(),
				Action:             a.recordList,
			},
			{
				Name:               "update",
				Usage:              "Update a record (patch semantics)",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordMutationFlags(),
				Action:             a.recordUpdate,
			},
			{
				Name:               "upsert",
				Usage:              "Create or replace a record",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordMutationFlags(),
				Action:             a.recordUpsert,
			},
			{
				Name:               "delete",
				Usage:              "Delete a record (idempotent, requires --force)",
				CustomHelpTemplate: commandHelpTemplate,
				Flags:              recordDeleteFlags(),
				Action:             a.recordDelete,
			},
		},
	}
}

func (a *App) recordCreate(ctx context.Context, cmd *cli.Command) error {
	data, err := readPayload(cmd)
	if err != nil {
		return invalidArgError("records", "create", err)
	}

	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "create", err)
	}

	var resp api.CreateRecordResponse
	if err := a.client.Call(ctx, "records.create", &api.CreateRecordRequest{
		URI:  uri,
		Data: data,
	}, &resp); err != nil {
		return wrapRPCError("records", "create", uri, err)
	}

	if cmd.Bool("quiet") {
		return nil
	}

	return formatRawJSON(cmd, resp.Data)
}

func (a *App) recordGet(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "get", err)
	}

	var resp api.GetRecordResponse
	if err := a.client.Call(ctx, "records.get", &api.GetRecordRequest{
		URI:    uri,
		Fields: parseFields(cmd.String("fields")),
	}, &resp); err != nil {
		return wrapRPCError("records", "get", uri, err)
	}

	return formatRawJSON(cmd, resp.Data)
}

func (a *App) recordList(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "list", err)
	}

	var resp api.ListRecordsResponse
	if err := a.client.Call(ctx, "records.list", &api.ListRecordsRequest{
		URI:    uri,
		Filter: cmd.String("filter"),
		Fields: parseFields(cmd.String("fields")),
		Limit:  int(cmd.Int("limit")),
		Offset: int(cmd.Int("offset")),
	}, &resp); err != nil {
		return wrapRPCError("records", "list", uri, err)
	}

	items := make([]any, len(resp.Items))
	for i, raw := range resp.Items {
		var m map[string]any
		if jsonErr := json.Unmarshal(raw, &m); jsonErr != nil {
			return jsonErr
		}

		items[i] = m
	}

	return formatList(cmd, items)
}

func (a *App) recordUpdate(ctx context.Context, cmd *cli.Command) error {
	data, err := readPayload(cmd)
	if err != nil {
		return invalidArgError("records", "update", err)
	}

	if data == nil {
		return invalidArgError("records", "update", fmt.Errorf("update requires a payload (--json, --file, or stdin)"))
	}

	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "update", err)
	}

	var resp api.UpdateRecordResponse
	if err := a.client.Call(ctx, "records.update", &api.UpdateRecordRequest{
		URI:  uri,
		Data: data,
	}, &resp); err != nil {
		return wrapRPCError("records", "update", uri, err)
	}

	if cmd.Bool("quiet") {
		return nil
	}

	return formatRawJSON(cmd, resp.Data)
}

func (a *App) recordUpsert(ctx context.Context, cmd *cli.Command) error {
	data, err := readPayload(cmd)
	if err != nil {
		return invalidArgError("records", "upsert", err)
	}

	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "upsert", err)
	}

	var resp api.UpsertRecordResponse
	if err := a.client.Call(ctx, "records.upsert", &api.UpsertRecordRequest{
		URI:  uri,
		Data: data,
	}, &resp); err != nil {
		return wrapRPCError("records", "upsert", uri, err)
	}

	if cmd.Bool("quiet") {
		return nil
	}

	return formatRawJSON(cmd, resp.Data)
}

func (a *App) recordDelete(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "delete", err)
	}

	if err := a.client.Call(ctx, "records.delete", &api.DeleteRecordRequest{
		URI: uri,
	}, nil); err != nil {
		return wrapRPCError("records", "delete", uri, err)
	}

	if cmd.Bool("quiet") {
		return nil
	}

	return formatOne(cmd, map[string]string{
		"status": "deleted",
		"uri":    uri,
	})
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
