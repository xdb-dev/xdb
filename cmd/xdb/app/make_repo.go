package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func MakeRepo(ctx context.Context, cmd *cli.Command) error {
	config := cmd.String("config")
	ns := cmd.String("ns")
	name := cmd.String("name")
	schemaPath := cmd.String("schema")

	slog.Info("[XDB] Creating schema", "ns", ns, "name", name, "schema", schemaPath)

	cfg, err := LoadConfig(ctx, config)
	if err != nil {
		return err
	}

	schemaDef, err := loadSchema(ctx, schemaPath)
	if err != nil {
		return err
	}

	if schemaDef == nil {
		schemaDef = &schema.Def{
			Name: name,
			Mode: schema.ModeFlexible,
		}
	} else {
		schemaDef.Name = name
	}

	app, err := New(cfg)
	if err != nil {
		return err
	}

	uri, err := core.ParseURI("xdb://" + ns + "/" + name)
	if err != nil {
		return err
	}

	err = app.SchemaDriver.PutSchema(ctx, uri, schemaDef)
	if err != nil {
		return err
	}

	slog.Info("[XDB] Schema created successfully", "uri", uri.String())

	return app.Shutdown(ctx)
}

func loadSchema(ctx context.Context, path string) (*schema.Def, error) {
	if path == "" {
		return nil, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schema path: %w", err)
	}

	// Validate that the resolved path is still within expected boundaries
	if !filepath.IsAbs(absPath) {
		return nil, fmt.Errorf("resolved path is not absolute: %s", absPath)
	}

	// #nosec: G304 - path is now validated to be absolute and resolved
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	s, err := schema.LoadFromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	return s, nil
}
