package store

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// enforce wraps a Driver with schema policy: per-tuple validation,
// mode enforcement, dynamic evolution, required-field checks, and
// revision stamping with CAS on schema writes. It is the ONE place
// this policy exists — drivers store verbatim, and [New] installs it
// unconditionally, so a Store without enforcement is unrepresentable.
func enforce(next Driver) Driver {
	return &enforcer{next: next}
}

type enforcer struct {
	next Driver
}

// --- Reads pass through ---

func (e *enforcer) GetTuples(
	ctx context.Context,
	uris ...*core.URI,
) ([]*core.Tuple, error) {
	return e.next.GetTuples(ctx, uris...)
}

func (e *enforcer) ScanTuples(
	ctx context.Context,
	scope *core.URI,
) iter.Seq2[*core.Tuple, error] {
	return e.next.ScanTuples(ctx, scope)
}

func (e *enforcer) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	return e.next.GetSchema(ctx, uri)
}

func (e *enforcer) ScanSchemas(
	ctx context.Context,
	scope *core.URI,
) iter.Seq2[*schema.Def, error] {
	return e.next.ScanSchemas(ctx, scope)
}

func (e *enforcer) DeleteSchema(ctx context.Context, uri *core.URI) error {
	return e.next.DeleteSchema(ctx, uri)
}

func (e *enforcer) DropRecords(ctx context.Context, uri *core.URI) error {
	return e.next.DropRecords(ctx, uri)
}

// --- Record writes ---

// Apply checks the mutation against its schema, then forwards it.
// The facade feeds mutations one at a time, so stateful checks
// (merge-that-creates) see the effects of earlier mutations in the
// same batch.
func (e *enforcer) Apply(ctx context.Context, m Mutation) error {
	if err := e.check(ctx, m); err != nil {
		return err
	}
	return e.next.Apply(ctx, m)
}

// check validates one mutation against its schema. No schema means no
// policy.
func (e *enforcer) check(ctx context.Context, m Mutation) error {
	def, err := e.next.GetSchema(ctx, m.Path.SchemaURI())
	if errors.Is(err, core.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	switch m.Op {
	case OpCreate, OpPut:
		// Replace-family ops carry the record's full tuple set, so the
		// required check is self-contained.
		if err := checkRequired(def, m.Tuples); err != nil {
			return err
		}
		return e.validateOrEvolve(ctx, def, m.Tuples)

	case OpPatch:
		if err := e.validateOrEvolve(ctx, def, m.Tuples); err != nil {
			return err
		}
		// A merge can only add or overwrite tuples, so a record
		// satisfying Required keeps satisfying it. The only stateful
		// check is merge-that-creates: the first tuples of a record
		// must include every required field. With no required fields
		// that check is vacuous, so skip the existence scan entirely.
		if !hasRequiredFields(def) {
			return nil
		}
		exists, err := recordExists(ctx, e.next, m.Path)
		if err != nil {
			return err
		}
		if !exists {
			return checkRequired(def, m.Tuples)
		}
		return nil

	case OpDelete:
		// Deleting the whole record is fine; stripping a required
		// attr from it is not.
		for _, attr := range m.Attrs {
			if def.Fields[attr].Required {
				return fmt.Errorf(
					"%w: cannot delete required field %q",
					core.ErrSchemaViolation, attr,
				)
			}
		}
		return nil

	default:
		return fmt.Errorf("store: unknown op %s", m.Op)
	}
}

// validateOrEvolve type-checks tuples per the schema mode. In dynamic
// mode, undeclared attributes evolve the schema: the new fields are
// inferred and written back (revision bumped by CloneWithFields).
func (e *enforcer) validateOrEvolve(
	ctx context.Context,
	def *schema.Def,
	tuples []*core.Tuple,
) error {
	switch def.Mode {
	case schema.ModeStrict, schema.ModeFlexible:
		if err := schema.ValidateTuples(def, tuples); err != nil {
			return fmt.Errorf("%w: %w", core.ErrSchemaViolation, err)
		}
		return nil

	case schema.ModeDynamic:
		newFields, err := schema.EvolveDynamic(def, tuples)
		if err != nil {
			return fmt.Errorf("%w: %w", core.ErrSchemaViolation, err)
		}
		if len(newFields) == 0 {
			return nil
		}
		evolved := def.CloneWithFields(newFields)
		return e.next.PutSchema(ctx, evolved)

	default:
		return fmt.Errorf("%w: unknown mode %q", core.ErrSchemaViolation, def.Mode)
	}
}

// checkRequired wraps [schema.CheckRequired] with the store's sentinel.
func checkRequired(def *schema.Def, tuples []*core.Tuple) error {
	if err := schema.CheckRequired(def, tuples); err != nil {
		return fmt.Errorf("%w: %w", core.ErrSchemaViolation, err)
	}
	return nil
}

// hasRequiredFields reports whether the schema declares any required
// field. When it does not, merge-that-creates needs no existence check.
func hasRequiredFields(def *schema.Def) bool {
	for _, field := range def.Fields {
		if field.Required {
			return true
		}
	}
	return false
}

// --- Schema writes ---

// CreateSchema validates the definition, stamps revision 1, and forwards.
func (e *enforcer) CreateSchema(ctx context.Context, def *schema.Def) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("%w: %w", core.ErrSchemaViolation, err)
	}
	def.Revision = 1
	return e.next.CreateSchema(ctx, def)
}

// PutSchema applies update policy: the definition must exist, must be a
// compatible evolution, and must pass the revision CAS. The stamped
// definition is then stored. (The enforcer's own dynamic-evolve
// write-backs bypass this policy by calling the driver directly —
// their revision is already bumped from the current def.)
func (e *enforcer) PutSchema(ctx context.Context, def *schema.Def) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("%w: %w", core.ErrSchemaViolation, err)
	}

	existing, err := e.next.GetSchema(ctx, def.URI)
	if err != nil {
		return err
	}

	if updateErr := schema.ValidateUpdate(existing, def); updateErr != nil {
		return fmt.Errorf("%w: %w", core.ErrSchemaViolation, updateErr)
	}

	next, err := schema.NextRevision(existing.Revision, def.Revision)
	if err != nil {
		return err
	}
	def.Revision = next

	return e.next.PutSchema(ctx, def)
}
