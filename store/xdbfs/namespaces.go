package xdbfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
)

// GetNamespace checks if any schema exists in the given namespace.
func (s *Store) GetNamespace(_ context.Context, uri *core.URI) (*core.NS, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nsDir := s.nsDir(uri)

	if hasSchema(nsDir) {
		return uri.NS(), nil
	}

	return nil, store.ErrNotFound
}

// ListNamespaces lists unique namespaces derived from schema directories.
func (s *Store) ListNamespaces(
	_ context.Context,
	q *store.Query,
) (*store.Page[*core.NS], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("fsstore: read root: %w", err)
	}

	var namespaces []*core.NS
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}

		nsDir := filepath.Join(s.root, e.Name())
		if hasSchema(nsDir) {
			namespaces = append(namespaces, core.NewNS(e.Name()))
		}
	}

	sort.Slice(namespaces, func(i, j int) bool {
		return namespaces[i].String() < namespaces[j].String()
	})

	return store.Paginate(namespaces, q), nil
}

// hasSchema checks if any subdirectory contains a _schema.json file.
func hasSchema(nsDir string) bool {
	entries, err := os.ReadDir(nsDir)
	if err != nil {
		return false
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		schemaFile := filepath.Join(nsDir, e.Name(), schemaFileName)
		if _, err := os.Stat(schemaFile); err == nil {
			return true
		}
	}

	return false
}
