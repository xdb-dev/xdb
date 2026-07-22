package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/validate"
	"github.com/xdb-dev/xdb/core"
)

func (a *App) importCmd() *cli.Command {
	return &cli.Command{
		Name:               "import",
		Usage:              "Import records from NDJSON",
		Category:           "operations",
		CustomHelpTemplate: commandHelpTemplate,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "uri", Usage: "Target schema URI"},
			&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to NDJSON file"},
			&cli.BoolFlag{Name: "create-only", Usage: "Use create instead of upsert (existing records with different data are skipped)"},
			&cli.BoolFlag{Name: "quiet", Usage: "Suppress output"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: a.importRecords,
	}
}

func (a *App) exportCmd() *cli.Command {
	return &cli.Command{
		Name:               "export",
		Usage:              "Export records as NDJSON",
		Category:           "operations",
		CustomHelpTemplate: commandHelpTemplate,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "uri", Usage: "Schema URI"},
			&cli.StringFlag{Name: "fields", Usage: "Comma-separated field mask"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: a.exportRecords,
	}
}

func (a *App) importRecords(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "import", err)
	}

	var reader io.Reader

	fileFlag := cmd.String("file")
	if fileFlag != "" {
		path, pathErr := validate.FilePath(fileFlag)
		if pathErr != nil {
			return invalidArgError("records", "import", pathErr)
		}

		f, openErr := os.Open(path) // #nosec G304 - path validated above
		if openErr != nil {
			return invalidArgError("records", "import", fmt.Errorf("open file: %w", openErr))
		}
		defer func() { _ = f.Close() }()

		reader = f
	} else if !isTerminal(os.Stdin) {
		reader = os.Stdin
	} else {
		return invalidArgError("records", "import", fmt.Errorf("import requires input (--file or stdin)"))
	}

	createOnly := cmd.Bool("create-only")
	quiet := cmd.Bool("quiet")
	op := "upsert"
	if createOnly {
		op = "create"
	}

	scanner := bufio.NewScanner(reader)
	summary := importSummary{}
	lineNum := 0

	var importErr error

	for scanner.Scan() {
		lineNum++

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		recordURI, uriErr := extractRecordURI(uri, line, lineNum)
		if uriErr != nil {
			summary.Failed++
			summary.FirstErrorLine = lineNum
			importErr = invalidArgError("records", "import", uriErr)
			break
		}

		if createOnly {
			err = a.client.Call(ctx, "records.create", &api.CreateRecordRequest{
				URI:  recordURI,
				Data: json.RawMessage(line),
			}, nil)
		} else {
			err = a.client.Call(ctx, "records.upsert", &api.UpsertRecordRequest{
				URI:  recordURI,
				Data: json.RawMessage(line),
			}, nil)
		}

		if err != nil {
			wrapped := wrapRPCError("records", op, recordURI, err)

			// Under --create-only an existing divergent record is an
			// expected skip, not a failure: local data is kept.
			if createOnly && envelopeCode(wrapped) == CodeConflict {
				summary.Skipped++
				continue
			}

			summary.Failed++
			summary.FirstErrorLine = lineNum
			importErr = prefixLineError(wrapped, lineNum)
			break
		}

		summary.Imported++

		if !quiet && summary.Imported%100 == 0 {
			_, _ = fmt.Fprintf(cmd.Root().ErrWriter, "Imported %d...\n", summary.Imported)
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && importErr == nil {
		importErr = fmt.Errorf("read input: %w", scanErr)
	}

	if !quiet {
		if formatErr := formatOne(cmd, summary); formatErr != nil && importErr == nil {
			importErr = formatErr
		}
	}

	return importErr
}

// importSummary is the machine-readable import result printed to stdout.
type importSummary struct {
	Imported       int `json:"imported"`
	Skipped        int `json:"skipped"`
	Failed         int `json:"failed"`
	FirstErrorLine int `json:"first_error_line,omitempty"`
}

// envelopeCode returns the error envelope's code, or "" for other errors.
func envelopeCode(err error) string {
	var env *output.ErrorEnvelope
	if errors.As(err, &env) {
		return env.Code
	}

	return ""
}

// prefixLineError prefixes an import error's message with the 1-based input
// line number it came from. It copies the envelope rather than mutating it in
// place, so the shared error-envelope value is never shared/aliased.
func prefixLineError(err error, line int) error {
	if err == nil {
		return nil
	}

	var env *output.ErrorEnvelope
	if errors.As(err, &env) {
		copied := *env
		copied.Message = fmt.Sprintf("line %d: %s", line, env.Message)
		return &copied
	}

	return fmt.Errorf("line %d: %w", line, err)
}

func (a *App) exportRecords(ctx context.Context, cmd *cli.Command) error {
	rawURI, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "export", err)
	}

	uri, err := core.ParseURI(rawURI)
	if err != nil {
		return invalidArgError("records", "export", err)
	}

	format := output.Format(cmd.String("output"))
	if format == "" {
		format = output.FormatNDJSON
	}
	if format != output.FormatNDJSON && format != output.FormatJSON {
		return invalidArgError("records", "export", fmt.Errorf(
			"export supports -o ndjson (default) or -o json; got %q", format,
		))
	}

	switch uri.Depth() {
	case 3:
		return a.exportSingleRecord(ctx, cmd, rawURI, format)
	case 2:
		// Distinguish a missing schema from an empty one before listing.
		if schemaErr := a.client.Call(ctx, "schemas.get", &api.GetSchemaRequest{URI: rawURI}, nil); schemaErr != nil {
			return wrapRPCError("records", "export", rawURI, schemaErr)
		}
		return a.exportSchemaRecords(ctx, cmd, rawURI, format)
	default:
		return invalidArgError("records", "export", fmt.Errorf(
			"export requires a schema URI xdb://ns/schema or a record URI xdb://ns/schema/id; got %q",
			rawURI,
		))
	}
}

// exportSingleRecord emits exactly one record.
func (a *App) exportSingleRecord(
	ctx context.Context,
	cmd *cli.Command,
	uri string,
	format output.Format,
) error {
	var resp api.GetRecordResponse
	if err := a.client.Call(ctx, "records.get", &api.GetRecordRequest{
		URI:    uri,
		Fields: parseFields(cmd.String("fields")),
	}, &resp); err != nil {
		return wrapRPCError("records", "export", uri, err)
	}

	m, err := unmarshalPreserving(resp.Data)
	if err != nil {
		return err
	}

	return output.New(format).FormatList(cmd.Root().Writer, []any{m})
}

// exportSchemaRecords streams every record in the schema.
func (a *App) exportSchemaRecords(
	ctx context.Context,
	cmd *cli.Command,
	uri string,
	format output.Format,
) error {
	f := output.New(output.FormatNDJSON)
	jsonItems := []any{}
	offset := 0

	for {
		var resp api.ListRecordsResponse
		if listErr := a.client.Call(ctx, "records.list", &api.ListRecordsRequest{
			URI:    uri,
			Fields: parseFields(cmd.String("fields")),
			Limit:  100,
			Offset: offset,
		}, &resp); listErr != nil {
			return wrapRPCError("records", "export", uri, listErr)
		}

		for _, raw := range resp.Items {
			m, jsonErr := unmarshalPreserving(raw)
			if jsonErr != nil {
				return jsonErr
			}

			if format == output.FormatJSON {
				jsonItems = append(jsonItems, m)
				continue
			}

			if fmtErr := f.FormatOne(cmd.Root().Writer, m); fmtErr != nil {
				return fmtErr
			}
		}

		if resp.NextOffset == 0 {
			break
		}

		offset = resp.NextOffset
	}

	if format == output.FormatJSON {
		return output.New(output.FormatJSON).FormatList(cmd.Root().Writer, jsonItems)
	}

	return nil
}

// extractRecordURI parses a JSON line, extracts the id, and builds a record URI.
func extractRecordURI(baseURI string, line []byte, lineNum int) (string, error) {
	var obj map[string]any
	if err := json.Unmarshal(line, &obj); err != nil {
		return "", fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
	}

	id, ok := obj["_id"]
	if !ok {
		id, ok = obj["id"]
	}

	if !ok {
		return "", fmt.Errorf("line %d: missing _id or id field", lineNum)
	}

	idStr := fmt.Sprintf("%v", id)

	return fmt.Sprintf("%s/%s", baseURI, idStr), nil
}
