package xdbredis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// validateAndEvolve validates a record's tuples against its schema.
// For strict schemas, unknown fields and type mismatches are rejected.
// For dynamic schemas, unknown fields are inferred and persisted back to
// the schema. For flexible schemas (or when no schema exists), validation
// is skipped. Returns [store.ErrSchemaViolation] on validation failure.
func (s *Store) validateAndEvolve(ctx context.Context, record *core.Record) error {
	schemaURI := record.URI().SchemaURI()

	def, err := s.GetSchema(ctx, schemaURI)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return err
	}

	if def.Mode == schema.ModeFlexible {
		return nil
	}

	switch def.Mode {
	case schema.ModeStrict:
		if vErr := schema.ValidateTuples(def, record.Tuples()); vErr != nil {
			return fmt.Errorf("%w: %w", store.ErrSchemaViolation, vErr)
		}

	case schema.ModeDynamic:
		return s.evolveDynamic(ctx, schemaURI, def, record.Tuples())
	}

	return nil
}

// evolveDynamic validates known fields and adds newly seen fields to the
// schema. Known fields with wrong types are rejected; unknown fields are
// inferred and persisted back to Redis.
func (s *Store) evolveDynamic(
	ctx context.Context,
	schemaURI *core.URI,
	def *schema.Def,
	tuples []*core.Tuple,
) error {
	newFields, err := schema.EvolveDynamic(def, tuples)
	if err != nil {
		return fmt.Errorf("%w: %w", store.ErrSchemaViolation, err)
	}
	if len(newFields) == 0 {
		return nil
	}

	// Build an evolved copy — never mutate the fetched def in place.
	evolved := &schema.Def{
		URI:    def.URI,
		Mode:   def.Mode,
		Fields: make(map[string]schema.FieldDef, len(def.Fields)+len(newFields)),
	}
	for k, v := range def.Fields {
		evolved.Fields[k] = v
	}
	for k, v := range newFields {
		evolved.Fields[k] = v
	}

	data, err := json.Marshal(evolved)
	if err != nil {
		return fmt.Errorf("xdbredis: marshal schema: %w", err)
	}
	if err := s.client.Set(ctx, s.schemaKey(schemaURI), data, 0).Err(); err != nil {
		return fmt.Errorf("xdbredis: persist evolved schema: %w", err)
	}

	return nil
}
