package app

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/core"
)

// Remove deletes a resource by URI.
// This is a thin wrapper - business logic is in App.RemoveByURI().
func Remove(ctx context.Context, cmd *cli.Command) error {
	// 1. Parse arguments
	uriStr := cmd.Args().First()
	if uriStr == "" {
		return fmt.Errorf("URI argument required")
	}

	uri, err := core.ParseURI(uriStr)
	if err != nil {
		return fmt.Errorf("invalid URI: %w", err)
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

	// 3. Confirm deletion (CLI-specific logic)
	if !cmd.Bool("force") {
		fmt.Fprintf(os.Stderr, "Delete %s? (y/N): ", uri.String())
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)
		if response != "y" && response != "Y" {
			return fmt.Errorf("deletion cancelled")
		}
	}

	// 4. Call business logic
	err = app.RemoveByURI(ctx, uri)
	if err != nil {
		return err
	}

	// 5. Format and write output
	format := getOutputFormat(cmd)
	writer := NewOutputWriter(os.Stdout, format)

	return writer.Write(map[string]any{
		"uri":    uri.String(),
		"status": "deleted",
	})
}
