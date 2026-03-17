package xdbsqlite

import (
	"context"
	"encoding/json"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// resolveStrategy determines the storage strategy for a record based on its schema mode.
// Returns the schema def (nil if no schema exists) and whether to use the table strategy.
func (s *Store) resolveStrategy(
	ctx context.Context,
	q *xsql.Queries,
	uri *core.URI,
) (def *schema.Def, useTable bool, err error) {
	def, err = s.cachedSchema(ctx, q, uri)
	if err != nil {
		return nil, false, err
	}

	if def == nil || def.Mode == schema.ModeFlexible {
		return def, false, nil
	}

	// Strict and Dynamic both use column tables.
	return def, true, nil
}

// validateAndEvolve validates record tuples against the schema and,
// for dynamic schemas, evolves the schema with new fields.
// Returns [store.ErrSchemaViolation] on validation failure.
func (s *Store) validateAndEvolve(
	ctx context.Context,
	q *xsql.Queries,
	def *schema.Def,
	tuples []*core.Tuple,
) error {
	if def == nil || def.Mode == schema.ModeFlexible {
		return nil
	}

	switch def.Mode {
	case schema.ModeStrict:
		return validateStrict(def, tuples)

	case schema.ModeDynamic:
		return s.evolveDynamic(ctx, q, def, tuples)
	}

	return nil
}

// validateStrict validates tuples against a strict schema.
// Unknown fields and type mismatches are rejected.
func validateStrict(def *schema.Def, tuples []*core.Tuple) error {
	if err := schema.ValidateTuples(def, tuples); err != nil {
		return store.ErrSchemaViolation
	}
	return nil
}

// evolveDynamic validates known fields and adds new fields to the schema.
// Known fields with wrong types are rejected; unknown fields are inferred and added.
// The cached schema is invalidated so the next lookup re-reads from DB.
func (s *Store) evolveDynamic(
	ctx context.Context,
	q *xsql.Queries,
	def *schema.Def,
	tuples []*core.Tuple,
) error {
	var newFields []schema.FieldDef
	var newNames []string

	for _, tuple := range tuples {
		attr := tuple.Attr().String()
		field, ok := def.Fields[attr]

		if !ok {
			// Unknown field — infer from tuple value type.
			newFields = append(newFields, schema.FieldDef{
				Type: tuple.Value().Type().ID(),
			})
			newNames = append(newNames, attr)
			continue
		}

		// Known field — reject type mismatch.
		if field.Type != tuple.Value().Type().ID() {
			return store.ErrSchemaViolation
		}
	}

	if len(newFields) == 0 {
		return nil
	}

	// Build evolved copy — never mutate the cached original.
	evolved := &schema.Def{
		URI:    def.URI,
		Mode:   def.Mode,
		Fields: make(map[string]schema.FieldDef, len(def.Fields)+len(newFields)),
	}
	for k, v := range def.Fields {
		evolved.Fields[k] = v
	}

	tableName := columnTableName(def.URI)
	for i, name := range newNames {
		err := q.AddColumn(ctx, xsql.AddColumnParams{
			Table: tableName,
			Column: xsql.Column{
				Name: name,
				Type: xsql.SQLiteTypeName(string(newFields[i].Type)),
			},
		})
		if err != nil {
			return err
		}
		evolved.Fields[name] = newFields[i]
	}

	data, err := json.Marshal(evolved)
	if err != nil {
		return err
	}
	err = q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: def.URI.NS().String(),
		Schema:    def.URI.Schema().String(),
		Data:      data,
	})
	if err != nil {
		return err
	}

	// Invalidate cache — the committed schema will be re-read on next access.
	s.invalidateSchema(def.URI)

	return nil
}
