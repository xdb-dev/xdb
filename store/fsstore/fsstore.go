// Package fsstore provides a filesystem-backed implementation of [store.Store].
//
// Records and schemas are stored as JSON files on disk. The directory
// hierarchy mirrors the XDB data model:
//
//	<root>/<namespace>/<schema>/_schema.json    # schema definition
//	<root>/<namespace>/<schema>/<id>.json       # record data
//
// The store is thread-safe via [sync.RWMutex] for concurrent in-process
// access. It is suitable for local development, CLI tools, and
// configuration storage.
package fsstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/xdb-dev/xdb/encoding/xdbjson"
)

const (
	schemaFileName = "_schema.json"
	jsonExt        = ".json"
	dirPerm        = 0o755
	filePerm       = 0o644
)

// Options configures the filesystem store.
type Options struct {
	// Indent controls JSON pretty-printing.
	// Default: "  " (two spaces).
	// Set to a single space " " for compact output.
	Indent string

	// CompactJSON disables indentation when true.
	CompactJSON bool
}

func (o Options) withDefaults() Options {
	if !o.CompactJSON && o.Indent == "" {
		o.Indent = "  "
	}
	return o
}

// Store is a filesystem-backed implementation of [store.Store].
type Store struct {
	enc  *xdbjson.Encoder
	root string
	opts Options
	mu   sync.RWMutex
}

// New creates a new filesystem store rooted at the given directory.
// The directory is created if it does not exist.
func New(root string, opts Options) (*Store, error) {
	opts = opts.withDefaults()

	if err := os.MkdirAll(root, dirPerm); err != nil {
		return nil, fmt.Errorf("fsstore: create root directory: %w", err)
	}

	enc := xdbjson.NewEncoder(xdbjson.Options{
		IncludeNS:     true,
		IncludeSchema: true,
	})

	return &Store{
		root: root,
		enc:  enc,
		opts: opts,
	}, nil
}

// Root returns the root directory of the store.
func (s *Store) Root() string {
	return s.root
}

// Health checks that the root directory exists and is writable.
func (s *Store) Health(_ context.Context) error {
	info, err := os.Stat(s.root)
	if err != nil {
		return fmt.Errorf("fsstore: root directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("fsstore: root is not a directory: %s", s.root)
	}

	// Check write access by creating and removing a temp file.
	f, err := os.CreateTemp(s.root, ".health-*")
	if err != nil {
		return fmt.Errorf("fsstore: root not writable: %w", err)
	}

	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)

	return nil
}

// writeFileAtomic writes data to path atomically by writing to a temp file
// first, then renaming.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)

	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}

	tmpName := f.Name()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpName)
		return err
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpName)
		return err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, path)
}
