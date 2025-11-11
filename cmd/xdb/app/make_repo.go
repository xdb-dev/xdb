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
	name := cmd.String("name")
	schemaPath := cmd.String("schema")

	slog.Info("[XDB] Making repo", "name", name, "schema", schemaPath)

	cfg, err := LoadConfig(ctx, config)
	if err != nil {
		return err
	}

	schema, err := loadSchema(ctx, schemaPath)
	if err != nil {
		return err
	}

	app, err := New(cfg)
	if err != nil {
		return err
	}

	var repo *core.Repo
	if schema != nil {
		schema.Name = name
		repo, err = core.NewRepo(schema)
	} else {
		repo, err = core.NewRepo(name)
	}
	if err != nil {
		return err
	}

	err = app.RepoDriver.MakeRepo(ctx, repo)
	if err != nil {
		return err
	}

	return app.Shutdown(ctx)
}

func loadSchema(ctx context.Context, path string) (*core.Schema, error) {
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
