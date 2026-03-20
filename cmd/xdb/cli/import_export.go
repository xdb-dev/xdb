package cli

import (
	"bufio"
	"context"
	"encoding/json"
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
			&cli.StringFlag{Name: "uri", Usage: "Target schema URI", Required: true},
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
			&cli.StringFlag{Name: "uri", Usage: "Schema URI", Required: true},
			&cli.StringFlag{Name: "fields", Usage: "Comma-separated field mask"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: a.exportRecords,
	}
}

func (a *App) importRecords(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return err
	}

	var reader io.Reader

	fileFlag := cmd.String("file")
	if fileFlag != "" {
		f, openErr := os.Open(fileFlag)
		if openErr != nil {
			return fmt.Errorf("open file: %w", openErr)
		}
		defer func() { _ = f.Close() }()

		reader = f
	} else if !isTerminal(os.Stdin) {
		reader = os.Stdin
	} else {
		return fmt.Errorf("import requires input (--file or stdin)")
	}

	createOnly := cmd.Bool("create-only")
	scanner := bufio.NewScanner(reader)
	imported := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		recordURI, err := extractRecordURI(uri, line, imported+1)
		if err != nil {
			return err
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
			return fmt.Errorf("line %d: %w", imported+1, err)
		}

		imported++

		if imported%100 == 0 {
			fmt.Fprintf(os.Stderr, "Imported %d...\n", imported)
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("read input: %w", scanErr)
	}

	fmt.Fprintf(os.Stderr, "Imported %d records\n", imported)

	return nil
}

func (a *App) exportRecords(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return err
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
			return listErr
		}

		for _, raw := range resp.Items {
			var m map[string]any
			if jsonErr := json.Unmarshal(raw, &m); jsonErr != nil {
				return jsonErr
			}

			if fmtErr := f.FormatOne(os.Stdout, m); fmtErr != nil {
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
