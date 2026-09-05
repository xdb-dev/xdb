package store

import (
	"context"
	"fmt"
	"time"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// now is the clock used to stamp [schema.FieldUpdated]. It is a var so
// tests can pin it.
var now = time.Now

// versioned wraps next so every record write stamps the record's system
// metadata: [schema.FieldVersion], a counter starting at 1, and
// [schema.FieldUpdated], the write's timestamp.
//
// It sits BELOW enforcement and above the raw driver. That placement is
// deliberate: enforcement validates what the caller actually wrote,
// against a definition already stamped with the system fields, and the
// stamped tuples are appended afterwards — so they are stored like any
// other declared field on every backend, and a driver needs to know
// nothing about versioning.
//
// A write can carry [schema.FieldVersion] as an optimistic-concurrency
// precondition. It is removed from the tuple set here and compared with
// the stored version by [schema.NextRevision]: equal bumps, stale
// returns [core.ErrConflict], absent (zero) writes unconditionally.
//
// Reads need no involvement — the system fields are ordinary tuples, so
// they flow back through scans, point reads, and filter pushdown alike.
func versioned(next Driver) Driver {
	return &versioner{Driver: next}
}

// versioner stamps record writes; reads and schema verbs pass through
// the embedded [Driver] untouched.
type versioner struct {
	Driver
}

// Apply stamps the mutation's system fields and forwards it.
func (v *versioner) Apply(ctx context.Context, m Mutation) error {
	if m.Op == OpDelete {
		return v.applyDelete(ctx, m)
	}
	return v.applyWrite(ctx, m)
}

// applyWrite handles every op except delete (patch, create, and put):
// take the precondition out of the tuple set, run the CAS against the
// stored version, and stamp the next version onto the forwarded
// mutation.
func (v *versioner) applyWrite(ctx context.Context, m Mutation) error {
	tuples, want := splitVersion(m.Tuples)

	// A patch whose only content was the precondition carries no facts.
	if len(tuples) == 0 && m.Op == OpPatch {
		return nil
	}

	// An empty put removes the record; stamping would resurrect it as a
	// husk of system tuples.
	if len(tuples) == 0 && m.Op == OpPut {
		return v.Driver.Apply(ctx, m)
	}

	// A create's target must not exist, so its stored version is 0 by
	// definition — the driver rejects the race, atomically.
	var cur int64
	if m.Op != OpCreate {
		var err error
		if cur, err = v.current(ctx, m.Path); err != nil {
			return err
		}
	}

	next, err := schema.NextRevision(cur, want)
	if err != nil {
		return err
	}

	m.Tuples = append(tuples, stampTuples(m.Path, next)...)

	return v.Driver.Apply(ctx, m)
}

// applyDelete handles removals. A whole-record delete needs no stamp —
// the version dies with the record. An attr delete bumps the version,
// unless it removed the record's last user tuple, in which case the
// record is gone and its system tuples must go with it.
func (v *versioner) applyDelete(ctx context.Context, m Mutation) error {
	if len(m.Attrs) == 0 {
		return v.Driver.Apply(ctx, m)
	}

	if err := v.Driver.Apply(ctx, m); err != nil {
		return err
	}

	remaining, err := v.userTuples(ctx, m.Path)
	if err != nil {
		return err
	}
	if remaining == 0 {
		return v.Driver.Apply(ctx, Mutation{Path: m.Path, Op: OpDelete})
	}

	cur, err := v.current(ctx, m.Path)
	if err != nil {
		return err
	}

	return v.Driver.Apply(ctx, Mutation{
		Path:   m.Path,
		Op:     OpPatch,
		Tuples: stampTuples(m.Path, cur+1),
	})
}

// current reads the record's stored version. An absent record, or one
// written before versioning, reads as 0 — which [schema.NextRevision]
// treats as "no base", so the next write starts the sequence.
func (v *versioner) current(ctx context.Context, path *core.URI) (int64, error) {
	uri, err := core.ParseURI(path.String() + "#" + schema.FieldVersion)
	if err != nil {
		return 0, err
	}

	tuples, err := v.GetTuples(ctx, uri)
	if err != nil {
		return 0, err
	}
	if len(tuples) == 0 {
		return 0, nil
	}

	return tuples[0].AsInt()
}

// userTuples counts the record's non-system tuples. A record exists
// exactly while it holds user facts; system tuples alone are a husk.
func (v *versioner) userTuples(ctx context.Context, path *core.URI) (int, error) {
	count := 0
	for tuple, err := range v.ScanTuples(ctx, path) {
		if err != nil {
			return 0, err
		}
		if !schema.IsSystemField(tuple.Attr()) {
			count++
		}
	}
	return count, nil
}

// splitVersion separates the optimistic-concurrency precondition from
// the tuples to write. A non-integer value reads as 0 (unconditional).
// When the record has a schema, enforcement has already type-checked
// the attr against the stamped definition, so a malformed precondition
// is rejected before it gets here. A schema-free record has no such
// check, so its malformed precondition is silently unconditional.
func splitVersion(tuples []*core.Tuple) ([]*core.Tuple, int64) {
	var want int64

	kept := make([]*core.Tuple, 0, len(tuples))
	for _, tuple := range tuples {
		if tuple.Attr() != schema.FieldVersion {
			kept = append(kept, tuple)
			continue
		}
		if v, err := tuple.AsInt(); err == nil {
			want = v
		}
	}

	return kept, want
}

// stampTuples builds the system tuples written with every record write.
func stampTuples(path *core.URI, version int64) []*core.Tuple {
	return []*core.Tuple{
		core.NewTuple(path.RecordPath(), schema.FieldVersion, version),
		core.NewTuple(path.RecordPath(), schema.FieldUpdated, now()),
	}
}

// normalizeDerived strips the attrs the store derives and rejects the
// ones a caller must not touch.
//
// [schema.FieldUpdated] is stamped on every write and [schema.FieldID]
// is projected from the path, so a caller echoing back a record it just
// read is normal and its values are simply dropped. An id that
// disagrees with the URI is not an echo but a misaddressed write, and a
// delete aimed at a system attr would corrupt the record's metadata;
// both are refused.
func normalizeDerived(m Mutation) (Mutation, error) {
	for _, attr := range m.Attrs {
		if schema.IsSystemField(attr) {
			return m, fmt.Errorf("%w: %q is maintained by the store and cannot be deleted",
				core.ErrSchemaViolation, attr)
		}
	}

	kept := make([]*core.Tuple, 0, len(m.Tuples))
	for _, tuple := range m.Tuples {
		switch tuple.Attr() {
		case schema.FieldUpdated:
			continue

		case schema.FieldID:
			id, err := tuple.AsStr()
			if err != nil || id != m.Path.ID() {
				return m, fmt.Errorf("%w: %q is %q for this record, not %v",
					core.ErrSchemaViolation, schema.FieldID, m.Path.ID(), tuple.Value())
			}
			continue

		default:
			kept = append(kept, tuple)
		}
	}

	m.Tuples = kept

	return m, nil
}
