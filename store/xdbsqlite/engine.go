package xdbsqlite

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// recordScanBatch is the page size for full-table scans. It is a var,
// not a const, so tests can shrink it to exercise paging boundaries.
var recordScanBatch = 1000

// isNoTable reports whether err signals a missing backing table. Reads
// treat a missing table as an empty one.
func isNoTable(err error) bool {
	return errors.Is(err, xsql.ErrNoTable)
}

// engine is one storage layout: how a single schema's records map to
// SQLite. The KV engine stores one row per attribute; the table engine
// stores one row per record, one column per field. Both are bound at
// construction to a query handle (database or transaction) and to the
// governing def — nil only for the KV engine, on schema-free records.
//
// Engines speak tuples and URIs. Op semantics (patch/create/put/
// delete) are not theirs: [runMutation] executes them once over these
// primitives.
type engine interface {
	// readRecord returns the record's full tuple set at path, or nil
	// when the record does not exist. A missing backing table reads as
	// absent.
	readRecord(ctx context.Context, path *core.URI) ([]*core.Tuple, error)

	// exists reports whether any tuples exist at path.
	exists(ctx context.Context, path *core.URI) (bool, error)

	// writeRecord replaces the record's full tuple set. An empty set
	// removes the record — row-exists ⇔ has-tuples.
	writeRecord(ctx context.Context, path *core.URI, tuples []*core.Tuple) error

	// scanRecords yields each record under the schema as its full tuple
	// set, in _id order, paged internally.
	scanRecords(ctx context.Context, schemaURI *core.URI) iter.Seq2[[]*core.Tuple, error]

	// queryRecords pushes a filtered, paginated list down to SQL.
	queryRecords(ctx context.Context, q *store.Query) (*store.Page[[]*core.Tuple], error)

	// ensure creates the backing storage if absent (idempotent DDL).
	ensure(ctx context.Context, schemaURI *core.URI) error

	// evolve alters backing storage after a def change; old is the
	// previously stored def.
	evolve(ctx context.Context, old *schema.Def) error

	// drop removes the backing storage and all its records.
	drop(ctx context.Context, schemaURI *core.URI) error
}

// engineFor picks the storage layout for a def: strict and dynamic
// defs use the table engine; flexible defs and schema-free records
// (def == nil) use the KV engine. This is the single routing point in
// the package.
func engineFor(q *xsql.Queries, def *schema.Def) engine {
	if def == nil || def.Mode == schema.ModeFlexible {
		return &kvEngine{q: q, def: def}
	}
	return &tableEngine{q: q, def: def}
}

// runMutation executes one mutation against an engine. The four-op
// semantics are identical across layouts — only the engine's
// exists/read/write differ.
//
// Backing storage is created lazily: reads treat a missing table as
// empty, and writeFull creates the table on demand if an insert hits
// one (schema-free first write, or a write after DropRecords). Keeping
// the DDL off the happy path avoids a CREATE TABLE on every mutation.
func runMutation(ctx context.Context, eng engine, m store.Mutation) error {
	exists, err := eng.exists(ctx, m.Path)
	if err != nil {
		return err
	}

	read := func() ([]*core.Tuple, error) {
		return eng.readRecord(ctx, m.Path)
	}
	writeFull := func(tuples []*core.Tuple) error {
		err := eng.writeRecord(ctx, m.Path, tuples)
		if !isNoTable(err) {
			return err
		}
		// Deleting from a missing table is already a no-op.
		if len(tuples) == 0 {
			return nil
		}
		if err := eng.ensure(ctx, m.Path.SchemaURI()); err != nil {
			return err
		}
		return eng.writeRecord(ctx, m.Path, tuples)
	}

	switch m.Op {
	case store.OpPatch:
		if !exists {
			return writeFull(m.Tuples)
		}
		current, err := read()
		if err != nil {
			return err
		}
		return writeFull(store.MergeTuples(current, m.Tuples))

	case store.OpCreate:
		if exists {
			return core.ErrAlreadyExists
		}
		return writeFull(m.Tuples)

	case store.OpPut:
		return writeFull(m.Tuples)

	case store.OpDelete:
		if len(m.Attrs) == 0 {
			return writeFull(nil)
		}
		if !exists {
			return nil
		}
		current, err := read()
		if err != nil {
			return err
		}
		return writeFull(store.RemoveAttrs(current, m.Attrs))

	default:
		return fmt.Errorf("xdbsqlite: unknown op %s", m.Op)
	}
}
