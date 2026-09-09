package xdbfs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// --- Definition reads ---

// GetSchema retrieves a definition by URI (ns + schema).
// Returns [core.ErrNotFound] if absent.
func (d *Driver) GetSchema(_ context.Context, uri *core.URI) (*schema.Def, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	def, err := readSchemaFile(d.schemaPath(uri))
	if err != nil {
		if isNotExist(err) {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return def, nil
}

// ScanSchemas yields definitions under scope (nil for all, or a namespace
// URI), in sorted path order.
func (d *Driver) ScanSchemas(
	_ context.Context,
	scope *core.URI,
) iter.Seq2[*schema.Def, error] {
	d.mu.RLock()
	snapshot, err := d.collectDefs(scope)
	d.mu.RUnlock()

	return func(yield func(*schema.Def, error) bool) {
		if err != nil {
			yield(nil, err)
			return
		}
		for _, def := range snapshot {
			if !yield(def, nil) {
				return
			}
		}
	}
}

// collectDefs snapshots all defs under scope by reading each schema
// directory's _schema.json. Callers must hold d.mu.
func (d *Driver) collectDefs(scope *core.URI) ([]*schema.Def, error) {
	uris, err := d.schemaDirsUnder(scope)
	if err != nil {
		return nil, err
	}

	var defs []*schema.Def
	for _, uri := range uris {
		def, err := readSchemaFile(d.schemaPath(uri))
		if err != nil {
			if isNotExist(err) {
				continue
			}
			return nil, err
		}
		defs = append(defs, def)
	}
	return defs, nil
}

// --- Definition writes ---

// CreateSchema stores a new definition verbatim. Returns
// [core.ErrAlreadyExists] if one exists; the exists-check is atomic
// via O_CREATE|O_EXCL.
func (d *Driver) CreateSchema(_ context.Context, def *schema.Def) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.marshalDef(def)
	if err != nil {
		return err
	}

	path := d.schemaPath(def.URI)
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("[xdb/xdbfs] create schema dir: %w", err)
	}

	if err := writeFileExclusive(path, data); err != nil {
		if errors.Is(err, os.ErrExist) {
			return core.ErrAlreadyExists
		}
		return fmt.Errorf("[xdb/xdbfs] create schema: %w", err)
	}
	return nil
}

// PutSchema stores a definition verbatim, unconditionally (upsert).
func (d *Driver) PutSchema(_ context.Context, def *schema.Def) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.marshalDef(def)
	if err != nil {
		return err
	}

	path := d.schemaPath(def.URI)
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("[xdb/xdbfs] create schema dir: %w", err)
	}

	if err := writeFileAtomic(path, data); err != nil {
		return fmt.Errorf("[xdb/xdbfs] write schema: %w", err)
	}
	return nil
}

// DeleteSchema deletes a definition. Returns [core.ErrNotFound] if
// absent. Empty schema and namespace directories are cleaned up.
func (d *Driver) DeleteSchema(_ context.Context, uri *core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	path := d.schemaPath(uri)

	if err := os.Remove(path); err != nil {
		if isNotExist(err) {
			return core.ErrNotFound
		}
		return fmt.Errorf("[xdb/xdbfs] delete schema: %w", err)
	}

	removeIfEmpty(d.schemaDir(uri))
	removeIfEmpty(d.nsDir(uri))

	return nil
}

// DropRecords deletes all record files belonging to a schema,
// keeping the definition. No-op if no records exist.
func (d *Driver) DropRecords(_ context.Context, uri *core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	dir := d.schemaDir(uri)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if isNotExist(err) {
			return nil
		}
		return fmt.Errorf("[xdb/xdbfs] read schema dir: %w", err)
	}

	for _, e := range entries {
		if !isRecordFile(e) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("[xdb/xdbfs] delete record: %w", err)
		}
	}

	return nil
}

// --- File helpers ---

// readSchemaFile reads and unmarshals a _schema.json file. Missing
// files surface as [os.ErrNotExist] for callers to map.
func readSchemaFile(path string) (*schema.Def, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var def schema.Def
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("[xdb/xdbfs] unmarshal schema %s: %w", path, err)
	}

	return &def, nil
}

// marshalDef encodes a definition verbatim — no validation, no
// revision stamping.
func (d *Driver) marshalDef(def *schema.Def) ([]byte, error) {
	data, err := json.MarshalIndent(def, "", d.opts.Indent)
	if err != nil {
		return nil, fmt.Errorf("[xdb/xdbfs] marshal schema: %w", err)
	}
	return data, nil
}
