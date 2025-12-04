package app

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"
	"log/slog"

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

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	s, err := schema.LoadFromJSON(data)
	if err != nil {
		return nil, err
	}

	return s, nil
}
