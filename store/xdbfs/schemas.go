package xdbfs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// GetSchema retrieves a schema definition by URI.
func (s *Store) GetSchema(_ context.Context, uri *core.URI) (*schema.Def, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readSchema(uri)
}

// ListSchemas lists schemas, optionally scoped by namespace URI.
func (s *Store) ListSchemas(
	_ context.Context,
	q *store.Query,
) (*store.Page[*schema.Def], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uri := q.URI

	var nsDirs []string
	if uri != nil {
		nsDirs = []string{s.nsDir(uri)}
	} else {
		entries, err := os.ReadDir(s.root)
		if err != nil {
			return nil, fmt.Errorf("fsstore: read root: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				nsDirs = append(nsDirs, filepath.Join(s.root, e.Name()))
			}
		}
	}

	var defs []*schema.Def
	for _, nsDir := range nsDirs {
		schemaDirs, err := os.ReadDir(nsDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("fsstore: read namespace dir: %w", err)
		}

		for _, sd := range schemaDirs {
			if !sd.IsDir() {
				continue
			}

			schemaFile := filepath.Join(nsDir, sd.Name(), schemaFileName)
			def, err := readSchemaFile(schemaFile)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return nil, err
			}
			defs = append(defs, def)
		}
	}

	sort.Slice(defs, func(i, j int) bool {
		return defs[i].URI.Path() < defs[j].URI.Path()
	})

	return store.Paginate(defs, q), nil
}

// CreateSchema creates a new schema definition.
func (s *Store) CreateSchema(
	_ context.Context,
	uri *core.URI,
	def *schema.Def,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.schemaPath(uri)

	if _, err := os.Stat(path); err == nil {
		return store.ErrAlreadyExists
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("fsstore: create schema dir: %w", err)
	}

	return s.writeSchema(path, def)
}

// UpdateSchema updates an existing schema definition.
func (s *Store) UpdateSchema(
	_ context.Context,
	uri *core.URI,
	def *schema.Def,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.schemaPath(uri)

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store.ErrNotFound
		}
		return fmt.Errorf("fsstore: stat schema: %w", err)
	}

	return s.writeSchema(path, def)
}

// DeleteSchema deletes a schema by URI.
func (s *Store) DeleteSchema(_ context.Context, uri *core.URI) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.schemaPath(uri)

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store.ErrNotFound
		}
		return fmt.Errorf("fsstore: stat schema: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("fsstore: delete schema: %w", err)
	}

	// Clean up empty directories.
	s.cleanEmptyDirs(uri)

	return nil
}

// DeleteSchemaRecords deletes all record files belonging to a schema.
func (s *Store) DeleteSchemaRecords(_ context.Context, uri *core.URI) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.schemaDir(uri)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("fsstore: read schema dir: %w", err)
	}

	for _, e := range entries {
		if e.Name() == schemaFileName || e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("fsstore: delete record: %w", err)
		}
	}

	return nil
}

func (s *Store) readSchema(uri *core.URI) (*schema.Def, error) {
	path := s.schemaPath(uri)
	def, err := readSchemaFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	return def, nil
}

func readSchemaFile(path string) (*schema.Def, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var def schema.Def
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("fsstore: unmarshal schema %s: %w", path, err)
	}

	return &def, nil
}

func (s *Store) writeSchema(path string, def *schema.Def) error {
	data, err := json.MarshalIndent(def, "", s.opts.Indent)
	if err != nil {
		return fmt.Errorf("fsstore: marshal schema: %w", err)
	}

	return writeFileAtomic(path, data)
}

// cleanEmptyDirs removes the schema and namespace directories if they are empty.
func (s *Store) cleanEmptyDirs(uri *core.URI) {
	schemaDir := s.schemaDir(uri)
	removeIfEmpty(schemaDir)

	nsDir := s.nsDir(uri)
	removeIfEmpty(nsDir)
}

func removeIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		_ = os.Remove(dir)
	}
}
