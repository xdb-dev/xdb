package xdbsqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

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
// Returns the schema def to use for the subsequent write; in dynamic mode this
// is the evolved def with the new columns, otherwise the original def is returned.
// Returns [store.ErrSchemaViolation] on validation failure.
func (s *Store) validateAndEvolve(
	ctx context.Context,
	q *xsql.Queries,
	def *schema.Def,
	tuples []*core.Tuple,
) (*schema.Def, error) {
	if def == nil {
		return def, nil
	}

	// Required is a declared-field property enforced on the full record in
	// every mode. Today all write paths are full-record replaces, so this is
	// correct for Create/Update/Upsert.
	if err := schema.CheckRequired(def, tuples); err != nil {
		return nil, fmt.Errorf("%w: %w", store.ErrSchemaViolation, err)
	}

	switch def.Mode {
	case schema.ModeStrict, schema.ModeFlexible:
		if err := schema.ValidateTuples(def, tuples); err != nil {
			return nil, fmt.Errorf("%w: %w", store.ErrSchemaViolation, err)
		}
		return def, nil

	case schema.ModeDynamic:
		return s.evolveDynamic(ctx, q, def, tuples)
	}

	return def, nil
}

// evolveDynamic validates known fields and adds new fields to the schema.
// Known fields with wrong types are rejected; unknown fields are inferred and added.
// Returns the evolved def (or the original if no evolution was needed) and
// invalidates the cache so concurrent readers re-fetch.
func (s *Store) evolveDynamic(
	ctx context.Context,
	q *xsql.Queries,
	def *schema.Def,
	tuples []*core.Tuple,
) (*schema.Def, error) {
	newFields, err := schema.EvolveDynamic(def, tuples)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", store.ErrSchemaViolation, err)
	}
	if len(newFields) == 0 {
		return def, nil
	}

	// Apply DDL in deterministic (alphabetical) order.
	newNames := make([]string, 0, len(newFields))
	for name := range newFields {
		newNames = append(newNames, name)
	}
	sort.Strings(newNames)

	// Build evolved copy — never mutate the cached original.
	evolved := &schema.Def{
		URI:         def.URI,
		Description: def.Description,
		Mode:        def.Mode,
		Revision:    def.Revision,
		Annotations: def.Annotations,
		Fields:      make(map[string]schema.Field, len(def.Fields)+len(newFields)),
	}
	for k, v := range def.Fields {
		evolved.Fields[k] = v
	}

	tableName := columnTableName(def.URI)
	for _, name := range newNames {
		field := newFields[name]
		err = q.AddColumn(ctx, xsql.AddColumnParams{
			Table: tableName,
			Column: xsql.Column{
				Name: name,
				Type: xsql.SQLiteTypeName(field.Type.ID().String()),
			},
		})
		if err != nil {
			return nil, err
		}
		evolved.Fields[name] = field
	}

	data, err := json.Marshal(evolved)
	if err != nil {
		return nil, err
	}
	if err := q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: def.URI.NS(),
		Schema:    def.URI.Schema(),
		Data:      data,
	}); err != nil {
		return nil, err
	}

	s.invalidateSchema(def.URI)
	return evolved, nil
}
