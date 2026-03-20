package xdbfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
	"github.com/xdb-dev/xdb/filter"
	"github.com/xdb-dev/xdb/store"
)

// GetRecord retrieves a record by URI.
func (s *Store) GetRecord(_ context.Context, uri *core.URI) (*core.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readRecord(uri)
}

// ListRecords lists records scoped by the given URI.
func (s *Store) ListRecords(
	_ context.Context,
	q *store.Query,
) (*store.Page[*core.Record], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uri := q.URI

	ns := uri.NS()
	schemaScope := uri.Schema()

	var schemaDirs []schemaInfo
	if schemaScope != nil {
		schemaDirs = append(schemaDirs, schemaInfo{
			ns:     ns.String(),
			schema: schemaScope.String(),
			dir:    s.schemaDir(uri),
		})
	} else {
		nsDir := s.nsDir(uri)
		entries, err := os.ReadDir(nsDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return &store.Page[*core.Record]{Items: nil}, nil
			}
			return nil, fmt.Errorf("fsstore: read namespace dir: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				schemaDirs = append(schemaDirs, schemaInfo{
					ns:     ns.String(),
					schema: e.Name(),
					dir:    filepath.Join(nsDir, e.Name()),
				})
			}
		}
	}

	var records []*core.Record
	for _, si := range schemaDirs {
		recs, err := s.readRecordsInDir(si)
		if err != nil {
			return nil, err
		}
		records = append(records, recs...)
	}

	if q.Filter != "" {
		f, err := filter.Compile(q.Filter, nil)
		if err != nil {
			return nil, err
		}
		records, err = filter.Records(f, records)
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].URI().Path() < records[j].URI().Path()
	})

	return store.Paginate(records, q), nil
}

// CreateRecord creates a new record.
func (s *Store) CreateRecord(_ context.Context, record *core.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.recordPath(record.URI())

	if _, err := os.Stat(path); err == nil {
		return store.ErrAlreadyExists
	}

	return s.writeRecord(path, record)
}

// UpdateRecord updates an existing record.
func (s *Store) UpdateRecord(_ context.Context, record *core.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.recordPath(record.URI())

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store.ErrNotFound
		}
		return fmt.Errorf("fsstore: stat record: %w", err)
	}

	return s.writeRecord(path, record)
}

// UpsertRecord creates or updates a record unconditionally.
func (s *Store) UpsertRecord(_ context.Context, record *core.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.recordPath(record.URI())
	return s.writeRecord(path, record)
}

// DeleteRecord deletes a record by URI.
func (s *Store) DeleteRecord(_ context.Context, uri *core.URI) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.recordPath(uri)

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store.ErrNotFound
		}
		return fmt.Errorf("fsstore: stat record: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("fsstore: delete record: %w", err)
	}

	return nil
}

func (s *Store) readRecord(uri *core.URI) (*core.Record, error) {
	path := s.recordPath(uri)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("fsstore: read record: %w", err)
	}

	dec := xdbjson.NewDecoder(
		xdbjson.WithNS(uri.NS().String()),
		xdbjson.WithSchema(uri.Schema().String()),
	)

	record, err := dec.ToRecord(data)
	if err != nil {
		return nil, fmt.Errorf("fsstore: decode record %s: %w", path, err)
	}

	return record, nil
}

func (s *Store) writeRecord(path string, record *core.Record) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("fsstore: create record dir: %w", err)
	}

	data, err := s.enc.FromRecord(record, xdbjson.WithIndent("", s.opts.Indent))
	if err != nil {
		return fmt.Errorf("fsstore: encode record: %w", err)
	}

	return writeFileAtomic(path, data)
}

type schemaInfo struct {
	ns     string
	schema string
	dir    string
}

func (s *Store) readRecordsInDir(si schemaInfo) ([]*core.Record, error) {
	entries, err := os.ReadDir(si.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("fsstore: read schema dir: %w", err)
	}

	dec := xdbjson.NewDecoder(xdbjson.WithNS(si.ns), xdbjson.WithSchema(si.schema))

	var records []*core.Record
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, jsonExt) || name == schemaFileName {
			continue
		}

		data, err := os.ReadFile(filepath.Join(si.dir, name))
		if err != nil {
			return nil, fmt.Errorf("fsstore: read record %s: %w", name, err)
		}

		record, err := dec.ToRecord(data)
		if err != nil {
			return nil, fmt.Errorf("fsstore: decode record %s: %w", name, err)
		}

		records = append(records, record)
	}

	return records, nil
}
