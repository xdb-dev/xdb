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
			&cli.BoolFlag{Name: "create-only", Usage: "Use create instead of upsert"},
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
		f, openErr := os.Open(fileFlag)
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
	op := "upsert"
	if createOnly {
		op = "create"
	}

	scanner := bufio.NewScanner(reader)
	imported := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		recordURI, uriErr := extractRecordURI(uri, line, imported+1)
		if uriErr != nil {
			return invalidArgError("records", "import", uriErr)
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
			return prefixLineError(wrapRPCError("records", op, recordURI, err), imported+1)
		}

		imported++

		if imported%100 == 0 {
			_, _ = fmt.Fprintf(cmd.Root().ErrWriter, "Imported %d...\n", imported)
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("read input: %w", scanErr)
	}

	_, _ = fmt.Fprintf(cmd.Root().ErrWriter, "Imported %d records\n", imported)

	return nil
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
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("records", "export", err)
	}

	f := output.New(output.FormatNDJSON)
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
			var m map[string]any
			if jsonErr := json.Unmarshal(raw, &m); jsonErr != nil {
				return jsonErr
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
