package xdbfs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xdb-dev/xdb/codec"
	codecjson "github.com/xdb-dev/xdb/codec/json"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/x"
)

// FSDriver is a filesystem-based driver for XDB.
// It stores each schema as a .schema.json file and each record as a separate JSON file.
//
// By default, FSDriver uses restrictive permissions (0o750 for directories, 0o600 for files)
// to minimize security risks. Use WithSharedAccess() option for more permissive settings
// suitable for shared environments.
type FSDriver struct {
	root     string
	codec    codec.KVCodec
	mu       sync.RWMutex
	dirPerm  os.FileMode
	filePerm os.FileMode
}

// Option is a functional option for configuring FSDriver.
type Option func(*FSDriver)

// WithPermissions sets custom file and directory permissions.
// Security Note: Use restrictive permissions (e.g., 0o750/0o600) to prevent
// unauthorized access. More permissive settings (e.g., 0o755/0o644) may be
// appropriate for shared read access scenarios.
func WithPermissions(dirPerm, filePerm os.FileMode) Option {
	return func(d *FSDriver) {
		d.dirPerm = dirPerm
		d.filePerm = filePerm
	}
}

// WithSharedAccess configures the driver with permissions suitable for shared
// read access (0o755 for directories, 0o644 for files).
// This provides backward compatibility with previous versions but may not be
// appropriate for sensitive data.
func WithSharedAccess() Option {
	return WithPermissions(0o755, 0o644)
}

// New creates a new filesystem driver with the given root directory.
// By default, uses restrictive permissions (0o750 for directories, 0o600 for files).
// Use functional options to customize permissions as needed.
func New(root string, opts ...Option) (*FSDriver, error) {
	d := &FSDriver{
		root:     root,
		codec:    codecjson.New(),
		dirPerm:  0o750,
		filePerm: 0o600,
	}

	for _, opt := range opts {
		opt(d)
	}

	if err := os.MkdirAll(root, d.dirPerm); err != nil {
		return nil, err
	}

	return d, nil
}

// GetSchema returns the schema definition for the given URI.
func (d *FSDriver) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	path := d.schemaPath(uri.NS().String(), uri.Schema().String())

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, driver.ErrNotFound
	}

	// #nosec G304 -- path is constructed internally via sanitized components
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return schema.LoadFromJSON(data)
}

// ListSchemas returns all schema definitions in the given namespace.
func (d *FSDriver) ListSchemas(ctx context.Context, uri *core.URI) ([]*schema.Def, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	nsPath := d.nsPath(uri.NS().String())
	if _, err := os.Stat(nsPath); os.IsNotExist(err) {
		return []*schema.Def{}, nil
	}

	schemas := make([]*schema.Def, 0)

	err := filepath.Walk(nsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".schema.json") {
			return nil
		}

		// #nosec G304 -- path is constructed internally via sanitized components
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		s, err := schema.LoadFromJSON(data)
		if err != nil {
			return err
		}

		schemas = append(schemas, s)
		return nil
	})

	return schemas, err
}

// PutSchema saves the schema definition.
func (d *FSDriver) PutSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	path := d.schemaPath(uri.NS().String(), uri.Schema().String())

	existing, err := d.readSchema(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if existing != nil {
		if existing.Mode != def.Mode {
			return driver.ErrSchemaModeChanged
		}

		for _, newField := range def.Fields {
			if oldField := existing.GetField(newField.Name); oldField != nil {
				if !oldField.Type.Equals(newField.Type) {
					return driver.ErrFieldChangeType
				}
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), d.dirPerm); err != nil {
		return err
	}

	data, err := schema.WriteToJSON(def)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, d.filePerm)
}

// readSchema reads a schema from the filesystem without locking.
func (d *FSDriver) readSchema(path string) (*schema.Def, error) {
	// #nosec G304 -- path is constructed internally via sanitized components
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return schema.LoadFromJSON(data)
}

// getSchemaLocked retrieves a schema without acquiring locks.
// Must be called while holding at least a read lock.
func (d *FSDriver) getSchemaLocked(uri *core.URI) (*schema.Def, error) {
	path := d.schemaPath(uri.NS().String(), uri.Schema().String())

	def, err := d.readSchema(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, driver.ErrNotFound
		}
		return nil, err
	}

	return def, nil
}

// writeSchemaLocked writes a schema without acquiring locks.
// Must be called while holding the write lock.
func (d *FSDriver) writeSchemaLocked(uri *core.URI, def *schema.Def) error {
	path := d.schemaPath(uri.NS().String(), uri.Schema().String())

	if err := os.MkdirAll(filepath.Dir(path), d.dirPerm); err != nil {
		return err
	}

	data, err := schema.WriteToJSON(def)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, d.filePerm)
}

// validateTuplesLocked validates tuples against their schema.
// For dynamic schemas, it infers and persists new fields.
// Must be called while holding the write lock.
func (d *FSDriver) validateTuplesLocked(schemaURI *core.URI, tuples []*core.Tuple) error {
	schemaDef, err := d.getSchemaLocked(schemaURI)
	if err != nil {
		return err
	}

	if schemaDef.Mode == schema.ModeDynamic {
		newFields, err := schema.InferFields(schemaDef, tuples)
		if err != nil {
			return err
		}

		if len(newFields) > 0 {
			schemaDef.AddFields(newFields...)
			if err := d.writeSchemaLocked(schemaURI, schemaDef); err != nil {
				return err
			}
		}
	}

	return schema.ValidateTuples(schemaDef, tuples)
}

// DeleteSchema deletes the schema definition.
func (d *FSDriver) DeleteSchema(ctx context.Context, uri *core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	path := d.schemaPath(uri.NS().String(), uri.Schema().String())

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(path)
}

// GetTuples returns the tuples for the given URIs.
func (d *FSDriver) GetTuples(ctx context.Context, uris []*core.URI) ([]*core.Tuple, []*core.URI, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tuples := make([]*core.Tuple, 0, len(uris))
	missed := make([]*core.URI, 0, len(uris))

	for _, uri := range uris {
		recordPath := d.recordPath(uri.NS().String(), uri.Schema().String(), uri.ID().String())

		record, err := d.readRecord(recordPath, uri.NS().String(), uri.Schema().String(), uri.ID().String())
		if err != nil {
			if os.IsNotExist(err) {
				missed = append(missed, uri)
				continue
			}
			return nil, nil, err
		}

		tuple := record.Get(uri.Attr().String())
		if tuple == nil {
			missed = append(missed, uri)
			continue
		}

		tuples = append(tuples, tuple)
	}

	return tuples, missed, nil
}

// PutTuples saves the tuples.
func (d *FSDriver) PutTuples(ctx context.Context, tuples []*core.Tuple) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	grouped := x.GroupBy(tuples, func(t *core.Tuple) string {
		return t.SchemaURI().String()
	})

	for _, tuples := range grouped {
		if err := d.validateTuplesLocked(tuples[0].SchemaURI(), tuples); err != nil {
			return err
		}
	}

	recordMap := make(map[string]*core.Record)

	for _, tuple := range tuples {
		id := tuple.ID().String()
		ns := tuple.NS().String()
		schemaName := tuple.Schema().String()
		recordPath := d.recordPath(ns, schemaName, id)

		record, ok := recordMap[recordPath]
		if !ok {
			var err error
			record, err = d.readRecord(recordPath, ns, schemaName, id)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			if record == nil {
				record = core.NewRecord(ns, schemaName, id)
			}
			recordMap[recordPath] = record
		}

		record.Set(tuple.Attr().String(), tuple.Value().Unwrap())
	}

	for path, record := range recordMap {
		if err := d.writeRecord(path, record); err != nil {
			return err
		}
	}

	return nil
}

// DeleteTuples deletes the tuples for the given URIs.
func (d *FSDriver) DeleteTuples(ctx context.Context, uris []*core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	recordMap := make(map[string]*core.Record)
	attrsToDelete := make(map[string][]string)

	for _, uri := range uris {
		id := uri.ID().String()
		ns := uri.NS().String()
		schema := uri.Schema().String()
		recordPath := d.recordPath(ns, schema, id)

		if _, ok := recordMap[recordPath]; !ok {
			record, err := d.readRecord(recordPath, ns, schema, id)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return err
			}
			recordMap[recordPath] = record
		}

		attrsToDelete[recordPath] = append(attrsToDelete[recordPath], uri.Attr().String())
	}

	for path, attrs := range attrsToDelete {
		record := recordMap[path]
		for _, attr := range attrs {
			if tuple := record.Get(attr); tuple != nil {
				record.Set(attr, nil)
			}
		}

		if record.IsEmpty() {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		} else {
			if err := d.writeRecord(path, record); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetRecords returns the records for the given URIs.
func (d *FSDriver) GetRecords(ctx context.Context, uris []*core.URI) ([]*core.Record, []*core.URI, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	records := make([]*core.Record, 0, len(uris))
	missed := make([]*core.URI, 0, len(uris))

	for _, uri := range uris {
		recordPath := d.recordPath(uri.NS().String(), uri.Schema().String(), uri.ID().String())

		record, err := d.readRecord(recordPath, uri.NS().String(), uri.Schema().String(), uri.ID().String())
		if err != nil {
			if os.IsNotExist(err) {
				missed = append(missed, uri)
				continue
			}
			return nil, nil, err
		}

		if record.IsEmpty() {
			missed = append(missed, uri)
			continue
		}

		records = append(records, record)
	}

	return records, missed, nil
}

// PutRecords saves the records.
func (d *FSDriver) PutRecords(ctx context.Context, records []*core.Record) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	grouped := x.GroupBy(records, func(r *core.Record) string {
		return r.SchemaURI().String()
	})

	for _, records := range grouped {
		var allTuples []*core.Tuple
		for _, record := range records {
			allTuples = append(allTuples, record.Tuples()...)
		}

		if err := d.validateTuplesLocked(records[0].SchemaURI(), allTuples); err != nil {
			return err
		}
	}

	for _, record := range records {
		recordPath := d.recordPath(record.NS().String(), record.Schema().String(), record.ID().String())

		if err := d.writeRecord(recordPath, record); err != nil {
			return err
		}
	}

	return nil
}

// DeleteRecords deletes the records for the given URIs.
func (d *FSDriver) DeleteRecords(ctx context.Context, uris []*core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, uri := range uris {
		recordPath := d.recordPath(uri.NS().String(), uri.Schema().String(), uri.ID().String())

		if err := os.Remove(recordPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// nsPath returns the directory path for a namespace.
func (d *FSDriver) nsPath(ns string) string {
	return filepath.Join(d.root, ns)
}

// schemaPath returns the file path for a schema.
func (d *FSDriver) schemaPath(ns, schemaName string) string {
	return filepath.Join(d.root, ns, schemaName, ".schema.json")
}

// recordPath returns the file path for a record.
func (d *FSDriver) recordPath(ns, schemaName, id string) string {
	sanitized := strings.ReplaceAll(id, "/", "_")
	return filepath.Join(d.root, ns, schemaName, fmt.Sprintf("%s.json", sanitized))
}

// readRecord reads a record from the filesystem.
func (d *FSDriver) readRecord(path, ns, schemaName, id string) (*core.Record, error) {
	// #nosec G304 -- path is constructed internally via sanitized components
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var recordData map[string]json.RawMessage
	if err := json.Unmarshal(data, &recordData); err != nil {
		return nil, err
	}

	record := core.NewRecord(ns, schemaName, id)

	for attr, rawValue := range recordData {
		value, err := d.codec.DecodeValue(rawValue)
		if err != nil {
			return nil, err
		}
		if value != nil {
			record.Set(attr, value.Unwrap())
		}
	}

	return record, nil
}

// writeRecord writes a record to the filesystem.
func (d *FSDriver) writeRecord(path string, record *core.Record) error {
	if err := os.MkdirAll(filepath.Dir(path), d.dirPerm); err != nil {
		return err
	}

	recordData := make(map[string]json.RawMessage)

	for _, tuple := range record.Tuples() {
		if tuple.Value() != nil && !tuple.Value().IsNil() {
			encoded, err := d.codec.EncodeValue(tuple.Value())
			if err != nil {
				return err
			}
			recordData[tuple.Attr().String()] = encoded
		}
	}

	data, err := json.MarshalIndent(recordData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, d.filePerm)
}
