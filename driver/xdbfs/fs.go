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
)

// FSDriver is a filesystem-based driver for XDB.
// It stores each schema as a .schema.json file and each record as a separate JSON file.
type FSDriver struct {
	root  string
	codec codec.KVCodec
	mu    sync.RWMutex
}

// New creates a new filesystem driver with the given root directory.
func New(root string) (*FSDriver, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}

	return &FSDriver{
		root:  root,
		codec: codecjson.New(),
	}, nil
}

// GetSchema returns the schema definition for the given URI.
func (d *FSDriver) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	path := d.schemaPath(uri.NS().String(), uri.Schema().String())

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, driver.ErrNotFound
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return schema.LoadFromJSON(data)
}

// ListSchemas returns all schema definitions in the given namespace.
func (d *FSDriver) ListSchemas(ctx context.Context, ns *core.NS) ([]*schema.Def, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	nsPath := d.nsPath(ns.String())
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
func (d *FSDriver) PutSchema(ctx context.Context, ns *core.NS, def *schema.Def) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	path := d.schemaPath(ns.String(), def.Name)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := schema.WriteToJSON(def)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
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

	recordMap := make(map[string]*core.Record)

	for _, tuple := range tuples {
		id := tuple.ID().String()
		ns := tuple.NS().String()
		schema := tuple.Schema().String()
		recordPath := d.recordPath(ns, schema, id)

		record, ok := recordMap[recordPath]
		if !ok {
			var err error
			record, err = d.readRecord(recordPath, ns, schema, id)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			if record == nil {
				record = core.NewRecord(ns, schema, id)
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
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
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

	return os.WriteFile(path, data, 0644)
}
