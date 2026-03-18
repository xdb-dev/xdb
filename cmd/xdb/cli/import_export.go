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
		defer f.Close()

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

		var obj map[string]any
		if jsonErr := json.Unmarshal(line, &obj); jsonErr != nil {
			return fmt.Errorf("line %d: invalid JSON: %w", imported+1, jsonErr)
		}

		id, ok := obj["_id"]
		if !ok {
			id, ok = obj["id"]
		}

		if !ok {
			return fmt.Errorf("line %d: missing _id or id field", imported+1)
		}

		recordURI := fmt.Sprintf("%s/%v", uri, id)

		if createOnly {
			_, createErr := a.records.Create(ctx, &api.CreateRecordRequest{
				URI:  recordURI,
				Data: json.RawMessage(line),
			})
			if createErr != nil {
				return fmt.Errorf("line %d: %w", imported+1, createErr)
			}
		} else {
			_, upsertErr := a.records.Upsert(ctx, &api.UpsertRecordRequest{
				URI:  recordURI,
				Data: json.RawMessage(line),
			})
			if upsertErr != nil {
				return fmt.Errorf("line %d: %w", imported+1, upsertErr)
			}
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
		resp, listErr := a.records.List(ctx, &api.ListRecordsRequest{
			URI:    uri,
			Fields: parseFields(cmd.String("fields")),
			Limit:  100,
			Offset: offset,
		})
		if listErr != nil {
			return listErr
		}

		for _, rec := range resp.Items {
			m, mapErr := a.recordToMap(rec)
			if mapErr != nil {
				return mapErr
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
