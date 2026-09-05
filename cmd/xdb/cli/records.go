package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
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
				Usage:              "Create a new record (CONFLICT if it exists with other data)",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "[URI]",
				Flags:              recordMutationFlags(),
				Action:             a.recordCreate,
			},
			{
				Name:               "get",
				Usage:              "Retrieve a record by URI",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "[URI]",
				Flags:              recordReadFlags(),
				Action:             a.recordGet,
			},
			{
				Name:               "list",
				Usage:              "List records in a schema or a namespace",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "[URI]",
				Flags:              recordListFlags(),
				Action:             a.recordList,
			},
			{
				Name:               "update",
				Usage:              "Update a record (patch semantics)",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "[URI]",
				Flags:              recordMutationFlags(),
				Action:             a.recordUpdate,
			},
			{
				Name:               "upsert",
				Usage:              "Create or replace a record",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "[URI]",
				Flags:              recordMutationFlags(),
				Action:             a.recordUpsert,
			},
			{
				Name:               "delete",
				Usage:              "Delete a record or one attribute (requires --force)",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "[URI]",
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

	dryRun := cmd.Bool("dry-run")

	var resp api.CreateRecordResponse
	if err := a.client.Call(ctx, "records.create", &api.CreateRecordRequest{
		URI:    uri,
		Data:   data,
		DryRun: dryRun,
	}, &resp); err != nil {
		return wrapRPCError("records", "create", uri, err)
	}

	if dryRun && resp.DryRun == nil {
		return dryRunIgnoredError("records", "create", uri)
	}

	if cmd.Bool("quiet") {
		return nil
	}

	if dryRun {
		return formatDryRun(cmd, resp.DryRun, resp.Data)
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

	if cmd.Bool("quiet") {
		return nil
	}

	return formatRawJSON(cmd, resp.Data)
}

func (a *App) recordList(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "list", err)
	}

	req := &api.ListRecordsRequest{
		URI:    uri,
		Filter: cmd.String("filter"),
		Fields: parseFields(cmd.String("fields")),
		Limit:  int(cmd.Int("limit")),
		Offset: int(cmd.Int("offset")),
	}

	page := output.Page{Items: []any{}}

	for {
		var resp api.ListRecordsResponse
		if err := a.client.Call(ctx, "records.list", req, &resp); err != nil {
			return wrapRPCError("records", "list", uri, err)
		}

		for _, raw := range resp.Items {
			m, jsonErr := unmarshalPreserving(raw)
			if jsonErr != nil {
				return jsonErr
			}
			page.Items = append(page.Items, m)
		}

		page.Total = resp.Total
		page.NextOffset = resp.NextOffset

		if !cmd.Bool("page-all") || resp.NextOffset == 0 {
			break
		}
		req.Offset = resp.NextOffset
	}

	if cmd.Bool("page-all") {
		// Every page was fetched, so there is no next offset to report.
		page.NextOffset = 0
	}

	return formatPage(cmd, page)
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

	dryRun := cmd.Bool("dry-run")

	var resp api.UpdateRecordResponse
	if err := a.client.Call(ctx, "records.update", &api.UpdateRecordRequest{
		URI:    uri,
		Data:   data,
		DryRun: dryRun,
	}, &resp); err != nil {
		return wrapRPCError("records", "update", uri, err)
	}

	if dryRun && resp.DryRun == nil {
		return dryRunIgnoredError("records", "update", uri)
	}

	if cmd.Bool("quiet") {
		return nil
	}

	if dryRun {
		return formatDryRun(cmd, resp.DryRun, resp.Data)
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

	dryRun := cmd.Bool("dry-run")

	var resp api.UpsertRecordResponse
	if err := a.client.Call(ctx, "records.upsert", &api.UpsertRecordRequest{
		URI:    uri,
		Data:   data,
		DryRun: dryRun,
	}, &resp); err != nil {
		return wrapRPCError("records", "upsert", uri, err)
	}

	if dryRun && resp.DryRun == nil {
		return dryRunIgnoredError("records", "upsert", uri)
	}

	if cmd.Bool("quiet") {
		return nil
	}

	if dryRun {
		return formatDryRun(cmd, resp.DryRun, resp.Data)
	}

	return formatRawJSON(cmd, resp.Data)
}

func (a *App) recordDelete(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "delete", err)
	}

	if !cmd.Bool("force") {
		return invalidArgError("records", "delete", fmt.Errorf("delete requires --force to confirm"))
	}

	dryRun := cmd.Bool("dry-run")

	var resp api.DeleteRecordResponse
	if err := a.client.Call(ctx, "records.delete", &api.DeleteRecordRequest{
		URI:     uri,
		Version: int64(cmd.Int("if-version")),
		DryRun:  dryRun,
	}, &resp); err != nil {
		return wrapRPCError("records", "delete", uri, err)
	}

	if dryRun && resp.DryRun == nil {
		return dryRunIgnoredError("records", "delete", uri)
	}

	if cmd.Bool("quiet") {
		return nil
	}

	if dryRun {
		return formatDryRun(cmd, resp.DryRun, nil)
	}

	return formatOne(cmd, map[string]string{
		"status": "deleted",
		"uri":    uri,
	})
}

func recordMutationFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Record URI"},
		&cli.StringFlag{Name: "json", Usage: "Inline JSON payload"},
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to input file"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Validate without writing"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		&cli.BoolFlag{Name: "quiet", Usage: "Suppress output"},
	}
}

func recordReadFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Record URI"},
		&cli.StringFlag{Name: "fields", Usage: "Comma-separated field mask"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		&cli.BoolFlag{Name: "quiet", Usage: "Suppress output"},
	}
}

func recordListFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Schema URI, or namespace URI to list across schemas"},
		&cli.StringFlag{Name: "filter", Usage: "CEL filter expression"},
		&cli.StringFlag{Name: "fields", Usage: "Comma-separated field mask"},
		&cli.IntFlag{Name: "limit", Usage: "Max items per page"},
		&cli.IntFlag{Name: "offset", Usage: "Page offset"},
		&cli.BoolFlag{Name: "page-all", Usage: "Fetch every page, then print one combined list"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
	}
}

func recordDeleteFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "uri", Usage: "Record URI"},
		&cli.BoolFlag{Name: "force", Usage: "Confirm deletion"},
		&cli.IntFlag{Name: "if-version", Usage: "Delete only if the record is at this version"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Validate without deleting"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		&cli.BoolFlag{Name: "quiet", Usage: "Suppress output"},
	}
}
