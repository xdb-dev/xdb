package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"

	"github.com/xdb-dev/xdb/core"
)

// Put creates or updates a record.
// This is a thin wrapper - business logic is in App.PutRecord().
func Put(ctx context.Context, cmd *cli.Command) error {
	// 1. Parse arguments
	uriStr := cmd.Args().First()
	if uriStr == "" {
		return fmt.Errorf("URI argument required")
	}

	uri, err := core.ParseURI(uriStr)
	if err != nil {
		return fmt.Errorf("invalid URI: %w", err)
	}

	if uri.ID() == nil {
		return fmt.Errorf("URI must include record ID")
	}

	// 2. Initialize app
	cfg, err := LoadConfig(ctx, cmd.String("config"))
	if err != nil {
		return err
	}

	app, err := New(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := app.Shutdown(ctx); err != nil {
			slog.Error("failed to shutdown app", "error", err)
		}
	}()

	// 3. Read and parse input
	data, err := readInputData(cmd)
	if err != nil {
		return err
	}

	record, err := parseRecord(uri, data, cmd.String("format"))
	if err != nil {
		return err
	}

	// 4. Call business logic
	err = app.PutRecord(ctx, record)
	if err != nil {
		return err
	}

	// 5. Format and write output
	format := getOutputFormat(cmd)
	writer := NewOutputWriter(os.Stdout, format)

	return writer.Write(map[string]any{
		"uri":    uri.String(),
		"status": "created",
	})
}

func readInputData(cmd *cli.Command) ([]byte, error) {
	filePath := cmd.String("file")

	if filePath != "" {
		cleanPath, err := filepath.Abs(filePath)
		if err != nil {
			return nil, fmt.Errorf("invalid file path: %w", err)
		}
		return os.ReadFile(cleanPath) // #nosec G304 - path is cleaned via filepath.Abs
	}

	// Read from stdin
	return io.ReadAll(os.Stdin)
}

func parseRecord(uri *core.URI, data []byte, format string) (*core.Record, error) {
	var attrs map[string]any

	switch format {
	case "yaml":
		err := yaml.Unmarshal(data, &attrs)
		if err != nil {
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}
	case "json", "":
		err := json.Unmarshal(data, &attrs)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	record := core.NewRecord(
		uri.NS().String(),
		uri.Schema().String(),
		uri.ID().String(),
	)

	for attr, value := range attrs {
		record.Set(attr, value)
	}

	return record, nil
}
