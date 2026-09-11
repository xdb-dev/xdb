package store

import (
	"context"
	"fmt"
	"strconv"
	"time"

	xerrors "github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// now is the clock used to stamp [schema.FieldUpdated]. It is a var so
// tests can pin it.
var now = time.Now

// versioned adds version checks and system-field stamping to record writes.
// It runs beneath enforcement so caller-supplied values are validated before
// system tuples are added. Reads pass through to next.
//
// [New] runs the check and write in one transaction when next is backed by
// a [TxDriver]. Other backends can accept concurrent writes with the same
// expected version.
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

// applyWrite checks the expected version for patch, create, and put,
// then adds the next version and timestamp before forwarding the mutation.
func (v *versioner) applyWrite(ctx context.Context, m Mutation) error {
	tuples, want := splitVersion(m.Tuples)

	// An empty patch changes nothing, even if it carries a version precondition.
	if len(tuples) == 0 && m.Op == OpPatch {
		return nil
	}

	// An empty put removes the record without adding system tuples.
	if len(tuples) == 0 && m.Op == OpPut {
		return v.Driver.Apply(ctx, m)
	}

	// A create expects an absent record with version 0. The driver
	// checks existence atomically when it applies the mutation.
	var cur int64
	if m.Op != OpCreate {
		var err error
		if cur, err = v.current(ctx, m.Path); err != nil {
			return err
		}
	}

	next, err := schema.NextRevision(cur, want)
	if err != nil {
		return versionConflict(err, cur, want)
	}

	// tuples is the fresh slice splitVersion returned, not a view of
	// m.Tuples, so appending to it cannot disturb the caller's slice.
	m.Tuples = append(tuples, stampTuples(m.Path, next)...) //nolint:gocritic // tuples is a fresh slice

	return v.Driver.Apply(ctx, m)
}

// versionConflict tags a failed version precondition with the two versions
// that disagree and with the way out. A stored version of 0 means the
// record is absent, where the advice to re-read and retry does not apply.
func versionConflict(err error, cur, want int64) error {
	fix := "re-read the record and retry with the current _version"
	if cur == 0 {
		fix = "the record does not exist. Create it, or write it without a _version"
	}

	return xerrors.Wrap(err,
		"expected", strconv.FormatInt(want, 10),
		"got", strconv.FormatInt(cur, 10),
		"fix", fix,
	)
}

// applyDelete removes the requested tuples and stamps the remaining record.
// If no user tuples remain, it removes the system tuples too.
// Whole-record deletes pass through without stamping.
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

// current reads the stored version, returning 0 when the version tuple is
// absent. The next stamped write starts at version 1 in that case.
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

// userTuples counts non-system tuples to determine whether a record remains.
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

// stampTuples builds the version and timestamp tuples for a write.
func stampTuples(path *core.URI, version int64) []*core.Tuple {
	return []*core.Tuple{
		core.NewTuple(path.RecordPath(), schema.FieldVersion, version),
		core.NewTuple(path.RecordPath(), schema.FieldUpdated, now()),
	}
}

// normalizeDerived removes caller-supplied [schema.FieldUpdated] and
// [schema.FieldID] values because the store derives them. It rejects an ID
// that differs from the record URI and attempts to delete system attributes.
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
