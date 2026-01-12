package app

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gojekfarm/xtools/errors"
	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func MakeSchema(ctx context.Context, cmd *cli.Command) error {
	uriStr := cmd.StringArg("uri")
	config := cmd.String("config")
	schemaPath := cmd.String("schema")

	if uriStr == "" {
		return ErrURIRequired
	}

	uri, err := core.ParseURI(uriStr)
	if err != nil {
		return errors.Wrap(ErrInvalidURI, "uri", uriStr)
	}

	cfg, err := LoadConfig(config)
	if err != nil {
		return err
	}

	schemaDef, err := loadSchema(schemaPath)
	if err != nil {
		return err
	}

	if schemaDef == nil {
		schemaDef = &schema.Def{
			NS:   uri.NS(),
			Name: uri.Schema().String(),
			Mode: schema.ModeFlexible,
		}
	} else {
		if schemaDef.NS == nil {
			schemaDef.NS = uri.NS()
		}
		if schemaDef.Name == "" {
			schemaDef.Name = uri.Schema().String()
		}
	}

	app, err := New(cfg)
	if err != nil {
		return err
	}

	err = app.SchemaDriver.PutSchema(ctx, uri, schemaDef)
	if err != nil {
		return err
	}

	slog.Info("Schema created successfully", "uri", uri.String())

	return app.Shutdown(ctx)
}

func loadSchema(path string) (*schema.Def, error) {
	if path == "" {
		return nil, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrap(err, "path", path)
	}

	if !filepath.IsAbs(absPath) {
		return nil, errors.Wrap(ErrPathNotAbsolute, "path", path, "resolved", absPath)
	}

	// #nosec: G304 - path is now validated to be absolute and resolved
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, errors.Wrap(err, "path", absPath)
	}

	s, err := schema.LoadFromJSON(data)
	if err != nil {
		return nil, err
	}

	return s, nil
}
