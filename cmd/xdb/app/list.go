package app

import (
	"context"
	"log/slog"
	"os"

	"github.com/gojekfarm/xtools/errors"
	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/core"
)

// List lists resources matching the URI pattern.
// This is a thin wrapper - business logic is in App.ListByURI().
func List(ctx context.Context, cmd *cli.Command) error {
	// 1. Parse arguments
	patternStr := cmd.Args().First()
	if patternStr == "" {
		return ErrURIRequired
	}

	uri, err := core.ParseURI(patternStr)
	if err != nil {
		return errors.Wrap(ErrInvalidURI, "uri", patternStr)
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
	opts := ListOptions{
		Limit:  int(cmd.Int("limit")),
		Offset: int(cmd.Int("offset")),
	}

	data, err := app.ListByURI(ctx, uri, opts)
	if err != nil {
		return err
	}

	// 4. Format and write output
	format := getOutputFormat(cmd)
	writer := NewOutputWriter(os.Stdout, format)

	return writer.Write(data)
}
