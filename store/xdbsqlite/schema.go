package xdbsqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// SchemaTx holds a Queries instance for schema operations.
type SchemaTx struct {
	q *xsql.Queries
}

func (s *SchemaTx) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	data, err := s.q.GetSchema(ctx, xsql.GetSchemaParams{
		Namespace: uri.NS().String(),
		Schema:    uri.Schema().String(),
	})
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, store.ErrNotFound
	}

	var def schema.Def
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

func (s *SchemaTx) ListSchemas(ctx context.Context, q *store.Query) (*store.Page[*schema.Def], error) {
	uri := q.URI
	var ns *string
	if uri != nil {
		n := uri.NS().String()
		ns = &n
	}

	rows, err := s.q.ListSchemas(ctx, xsql.ListSchemasParams{
		Namespace: ns,
		Limit:     10000,
	})
	if err != nil {
		return nil, err
	}

	defs := make([]*schema.Def, 0, len(rows))
	for _, row := range rows {
		var def schema.Def
		if err := json.Unmarshal(row.Data, &def); err != nil {
			return nil, err
		}
		defs = append(defs, &def)
	}

	return store.Paginate(defs, q), nil
}

func (s *SchemaTx) CreateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	exists, err := s.q.SchemaExists(ctx, xsql.SchemaExistsParams{
		Namespace: uri.NS().String(),
		Schema:    uri.Schema().String(),
	})
	if err != nil {
		return err
	}
	if exists {
		return store.ErrAlreadyExists
	}

	data, err := json.Marshal(def)
	if err != nil {
		return err
	}

	if err := s.q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: uri.NS().String(),
		Schema:    uri.Schema().String(),
		Data:      data,
	}); err != nil {
		return err
	}

	// Create the backing table based on schema mode.
	switch def.Mode {
	case schema.ModeStrict, schema.ModeDynamic:
		return s.q.CreateTable(ctx, xsql.CreateTableParams{
			Table:   columnTableName(uri),
			Columns: columnDefs(def),
		})
	default: // Flexible
		return s.q.CreateKVTable(ctx, xsql.CreateKVTableParams{
			Table: kvTableName(uri),
		})
	}
}

func (s *SchemaTx) UpdateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	oldDef, err := s.GetSchema(ctx, uri)
	if err != nil {
		return err // includes ErrNotFound
	}

	if oldDef.Mode != def.Mode {
		return fmt.Errorf(
			"%w: cannot change schema mode from %s to %s",
			store.ErrSchemaViolation,
			oldDef.Mode,
			def.Mode,
		)
	}

	err = s.evolveSchema(ctx, uri, oldDef, def)
	if err != nil {
		return err
	}

	data, err := json.Marshal(def)
	if err != nil {
		return err
	}

	return s.q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: uri.NS().String(),
		Schema:    uri.Schema().String(),
		Data:      data,
	})
}

// evolveSchema alters the backing column table to match field changes
// between oldDef and newDef. Only applies to strict/dynamic modes.
func (s *SchemaTx) evolveSchema(
	ctx context.Context,
	uri *core.URI,
	oldDef *schema.Def,
	newDef *schema.Def,
) error {
	if newDef.Mode == schema.ModeFlexible {
		return nil
	}

	tableName := columnTableName(uri)

	// Check for type changes and add new columns.
	for name, newField := range newDef.Fields {
		oldField, exists := oldDef.Fields[name]
		if !exists {
			err := s.q.AddColumn(ctx, xsql.AddColumnParams{
				Table: tableName,
				Column: xsql.Column{
					Name: name,
					Type: xsql.SQLiteTypeName(string(newField.Type)),
				},
			})
			if err != nil {
				return err
			}
			continue
		}
		if oldField.Type != newField.Type {
			return fmt.Errorf(
				"%w: cannot change type of field %q from %s to %s",
				store.ErrSchemaViolation,
				name,
				oldField.Type,
				newField.Type,
			)
		}
	}

	// Drop removed columns.
	for name := range oldDef.Fields {
		if _, exists := newDef.Fields[name]; !exists {
			err := s.q.DropColumn(ctx, xsql.DropColumnParams{
				Table:  tableName,
				Column: name,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SchemaTx) DeleteSchema(ctx context.Context, uri *core.URI) error {
	exists, err := s.q.SchemaExists(ctx, xsql.SchemaExistsParams{
		Namespace: uri.NS().String(),
		Schema:    uri.Schema().String(),
	})
	if err != nil {
		return err
	}
	if !exists {
		return store.ErrNotFound
	}

	return s.q.DeleteSchema(ctx, xsql.DeleteSchemaParams{
		Namespace: uri.NS().String(),
		Schema:    uri.Schema().String(),
	})
}

// --- Store delegation ---

// GetSchema retrieves a schema definition by URI.
func (s *Store) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	stx := &SchemaTx{q: xsql.NewQueries(tx)}

	res, err := stx.GetSchema(ctx, uri)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return res, nil
}

// ListSchemas lists schemas, optionally scoped by namespace URI.
func (s *Store) ListSchemas(
	ctx context.Context,
	q *store.Query,
) (*store.Page[*schema.Def], error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	stx := &SchemaTx{q: xsql.NewQueries(tx)}

	res, err := stx.ListSchemas(ctx, q)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return res, nil
}

// CreateSchema creates a new schema definition.
func (s *Store) CreateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stx := &SchemaTx{q: xsql.NewQueries(tx)}

	if err := stx.CreateSchema(ctx, uri, def); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.cache.Store(cacheKey(uri), def)
	return nil
}

// UpdateSchema updates an existing schema definition. Mode changes are not allowed.
func (s *Store) UpdateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stx := &SchemaTx{q: xsql.NewQueries(tx)}

	if err := stx.UpdateSchema(ctx, uri, def); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.cache.Store(cacheKey(uri), def)
	return nil
}

// DeleteSchema deletes a schema by URI.
func (s *Store) DeleteSchema(ctx context.Context, uri *core.URI) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stx := &SchemaTx{q: xsql.NewQueries(tx)}

	if err := stx.DeleteSchema(ctx, uri); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.invalidateSchema(uri)
	return nil
}
