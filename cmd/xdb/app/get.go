package app

import (
	"context"
	"log/slog"
	"os"

	"github.com/gojekfarm/xtools/errors"
	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/core"
)

// Get retrieves a resource by URI.
// This is a thin wrapper - business logic is in App.GetByURI().
func Get(ctx context.Context, cmd *cli.Command) error {
	// 1. Parse arguments
	uriStr := cmd.Args().First()
	if uriStr == "" {
		return ErrURIRequired
	}

	uri, err := core.ParseURI(uriStr)
	if err != nil {
		return errors.Wrap(ErrInvalidURI, "uri", uriStr)
	}

	// 2. Initialize app
	cfg, err := LoadConfig(cmd.String("config"))
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

	// 3. Call business logic
	data, err := app.GetByURI(ctx, uri)
	if err != nil {
		return err
	}

	// 4. Format and write output
	format := getOutputFormat(cmd)
	writer := NewOutputWriter(os.Stdout, format)

	return writer.Write(data)
}

// getOutputFormat gets format from --output flag or auto-detects.
func getOutputFormat(cmd *cli.Command) Format {
	formatStr := cmd.String("output")
	if formatStr != "" {
		return Format(formatStr)
	}
	return SelectDefaultFormat()
}
