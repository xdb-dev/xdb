package xdbfs

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
	"github.com/xdb-dev/xdb/store"
)

// --- Tuple reads ---

// GetTuples returns the tuples at the given attr-level URIs.
// Absent attrs are omitted; results are in request order.
func (d *Driver) GetTuples(
	_ context.Context,
	uris ...*core.URI,
) ([]*core.Tuple, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Decode each record file at most once per call.
	cache := make(map[string]map[string]*core.Tuple)

	var got []*core.Tuple
	for _, uri := range uris {
		file := d.recordPath(uri)

		attrs, ok := cache[file]
		if !ok {
			tuples, err := d.readRecordTuples(uri)
			if err != nil {
				return nil, err
			}
			attrs = make(map[string]*core.Tuple, len(tuples))
			for _, tuple := range tuples {
				attrs[tuple.Attr()] = tuple
			}
			cache[file] = attrs
		}

		if tuple, ok := attrs[uri.Attr()]; ok {
			got = append(got, tuple)
		}
	}

	return got, nil
}

// ScanTuples yields every tuple under scope: a namespace, a schema, or
// a record path. Record files are visited in sorted path order, so
// tuples of one record are contiguous and the scan is deterministic.
func (d *Driver) ScanTuples(
	_ context.Context,
	scope *core.URI,
) iter.Seq2[*core.Tuple, error] {
	d.mu.RLock()
	snapshot, err := d.collectTuples(scope)
	d.mu.RUnlock()

	return func(yield func(*core.Tuple, error) bool) {
		if err != nil {
			yield(nil, err)
			return
		}
		for _, tuple := range snapshot {
			if !yield(tuple, nil) {
				return
			}
		}
	}
}

// collectTuples snapshots all tuples under scope. Callers must hold
// d.mu.
func (d *Driver) collectTuples(scope *core.URI) ([]*core.Tuple, error) {
	if scope != nil && scope.ID() != "" {
		return d.readRecordTuples(scope)
	}

	dirs, err := d.schemaDirsUnder(scope)
	if err != nil {
		return nil, err
	}

	var snapshot []*core.Tuple
	for _, uri := range dirs {
		tuples, err := d.readSchemaDirTuples(uri)
		if err != nil {
			return nil, err
		}
		snapshot = append(snapshot, tuples...)
	}
	return snapshot, nil
}

// schemaDirsUnder resolves scope (nil, namespace, or schema) to the
// schema URIs whose directories exist, in sorted path order.
func (d *Driver) schemaDirsUnder(scope *core.URI) ([]*core.URI, error) {
	if scope != nil && scope.Schema() != "" {
		return []*core.URI{scope.SchemaURI()}, nil
	}

	var nsNames []string
	if scope != nil {
		nsNames = []string{scope.NS()}
	} else {
		entries, err := os.ReadDir(d.root)
		if err != nil {
			return nil, fmt.Errorf("xdbfs: read root: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				nsNames = append(nsNames, e.Name())
			}
		}
	}

	var uris []*core.URI
	for _, ns := range nsNames {
		entries, err := os.ReadDir(filepath.Join(d.root, ns))
		if err != nil {
			if isNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("xdbfs: read namespace dir: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				uris = append(uris, core.MustNewURI(ns, e.Name()))
			}
		}
	}
	return uris, nil
}

// readSchemaDirTuples decodes every record file in one schema
// directory, in sorted file order.
func (d *Driver) readSchemaDirTuples(schemaURI *core.URI) ([]*core.Tuple, error) {
	dir := d.schemaDir(schemaURI)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if isNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("xdbfs: read schema dir: %w", err)
	}

	dec := d.newDecoder(schemaURI)

	var tuples []*core.Tuple
	for _, e := range entries {
		if !isRecordFile(e) {
			continue
		}
		fileTuples, err := decodeRecordFile(dec, filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		tuples = append(tuples, fileTuples...)
	}
	return tuples, nil
}

// isRecordFile reports whether a schema-directory entry is a record
// file (a .json file other than _schema.json).
func isRecordFile(e os.DirEntry) bool {
	name := e.Name()
	if e.IsDir() || name == schemaFileName {
		return false
	}
	return strings.HasSuffix(name, jsonExt)
}

// --- Tuple writes ---

// Apply executes one mutation atomically: files are replaced via
// temp-file + rename, and creates use O_EXCL.
func (d *Driver) Apply(_ context.Context, m store.Mutation) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.applyMutation(m)
}

// applyMutation dispatches one mutation per the op table in
// [store.Mutation].
func (d *Driver) applyMutation(m store.Mutation) error {
	switch m.Op {
	case store.OpPatch:
		return d.applyMerge(m)
	case store.OpCreate:
		return d.applyCreate(m)
	case store.OpPut:
		return d.writeOrRemove(m.Path, m.Tuples)
	case store.OpDelete:
		return d.applyDelete(m)
	default:
		return fmt.Errorf("xdbfs: unknown op %s", m.Op)
	}
}

// applyMerge reads the record file, overlays the mutation's tuples,
// and writes the result back. The record springs into existence on
// the first patch.
func (d *Driver) applyMerge(m store.Mutation) error {
	if len(m.Tuples) == 0 {
		return nil
	}

	existing, err := d.readRecordTuples(m.Path)
	if err != nil {
		return err
	}

	return d.writeRecord(m.Path, store.MergeTuples(existing, m.Tuples))
}

// applyCreate writes the record file with O_CREATE|O_EXCL so exactly
// one concurrent creator wins. Returns [core.ErrAlreadyExists] if the
// file exists.
func (d *Driver) applyCreate(m store.Mutation) error {
	file := d.recordPath(m.Path)

	// A record with zero tuples does not exist: nothing to write,
	// only the exists-check applies.
	if len(m.Tuples) == 0 {
		if _, err := os.Stat(file); err == nil {
			return core.ErrAlreadyExists
		}
		return nil
	}

	data, err := d.encodeRecord(m.Path, m.Tuples)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return fmt.Errorf("xdbfs: create record dir: %w", err)
	}

	if err := writeFileExclusive(file, data); err != nil {
		if errors.Is(err, os.ErrExist) {
			return core.ErrAlreadyExists
		}
		return fmt.Errorf("xdbfs: create record: %w", err)
	}
	return nil
}

// applyDelete removes the whole record file (empty Attrs) or the named
// tuples from it. Idempotent: absent records and attrs are no-ops.
func (d *Driver) applyDelete(m store.Mutation) error {
	if len(m.Attrs) == 0 {
		return removeRecordFile(d.recordPath(m.Path))
	}

	existing, err := d.readRecordTuples(m.Path)
	if err != nil || existing == nil {
		return err
	}

	return d.writeOrRemove(m.Path, store.RemoveAttrs(existing, m.Attrs))
}

// --- File helpers ---

// encodeRecord encodes a tuple set as a record JSON document.
func (d *Driver) encodeRecord(path *core.URI, tuples []*core.Tuple) ([]byte, error) {
	record := core.NewRecordFromTuples(path, tuples)

	data, err := d.enc.FromRecord(record, xdbjson.WithIndent("", d.opts.Indent))
	if err != nil {
		return nil, fmt.Errorf("xdbfs: encode record: %w", err)
	}
	return data, nil
}

// writeRecord writes the tuple set to the record file atomically
// (temp file + rename).
func (d *Driver) writeRecord(path *core.URI, tuples []*core.Tuple) error {
	data, err := d.encodeRecord(path, tuples)
	if err != nil {
		return err
	}

	file := d.recordPath(path)
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return fmt.Errorf("xdbfs: create record dir: %w", err)
	}

	if err := writeFileAtomic(file, data); err != nil {
		return fmt.Errorf("xdbfs: write record: %w", err)
	}
	return nil
}

// writeOrRemove replaces the record's tuple set. A record with zero
// tuples does not exist, so an empty set removes the file.
func (d *Driver) writeOrRemove(path *core.URI, tuples []*core.Tuple) error {
	if len(tuples) == 0 {
		return removeRecordFile(d.recordPath(path))
	}
	return d.writeRecord(path, tuples)
}

// removeRecordFile removes a record file, tolerating absence.
func removeRecordFile(file string) error {
	if err := os.Remove(file); err != nil && !isNotExist(err) {
		return fmt.Errorf("xdbfs: delete record: %w", err)
	}
	return nil
}

// newDecoder builds a decoder for records of one schema. When the
// schema's def file exists it guides type-aware decoding (typed
// arrays, times, bytes); otherwise number inference alone applies.
func (d *Driver) newDecoder(schemaURI *core.URI) *xdbjson.Decoder {
	opts := []xdbjson.Option{
		xdbjson.WithNS(schemaURI.NS()),
		xdbjson.WithSchema(schemaURI.Schema()),
		xdbjson.WithNumberInference(),
	}
	if def, err := readSchemaFile(d.schemaPath(schemaURI)); err == nil {
		opts = append(opts, xdbjson.WithDef(def))
	}
	return xdbjson.NewDecoder(opts...)
}

// readRecordTuples decodes a single record file into its tuples.
// Returns (nil, nil) if the file is absent.
func (d *Driver) readRecordTuples(path *core.URI) ([]*core.Tuple, error) {
	dec := d.newDecoder(path.SchemaURI())
	return decodeRecordFile(dec, d.recordPath(path))
}

// decodeRecordFile reads and decodes one record file. Returns
// (nil, nil) if the file is absent.
func decodeRecordFile(dec *xdbjson.Decoder, file string) ([]*core.Tuple, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		if isNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("xdbfs: read record: %w", err)
	}

	record, err := dec.ToRecord(data)
	if err != nil {
		return nil, fmt.Errorf("xdbfs: decode record %s: %w", file, err)
	}
	return record.Tuples(), nil
}
