// Package xdbfs provides a filesystem-backed implementation of
// [store.Driver].
//
// Records and schema definitions are stored as JSON files. The
// directory hierarchy mirrors the XDB data model:
//
//	<root>/<namespace>/<schema>/_schema.json    # schema definition
//	<root>/<namespace>/<schema>/<id>.json       # one file per record
//
// Use [store.New] to add schema enforcement and versioning:
//
//	d, err := xdbfs.NewDriver(root, xdbfs.Options{})
//	st := store.New(d)
//
// A [sync.RWMutex] makes the driver safe for concurrent in-process
// use. Record creation also uses O_EXCL, so the filesystem resolves
// create races. The driver is suitable for local development, CLI
// tools, and config storage.
package xdbfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	schemaFileName = "_schema.json"
	jsonExt        = ".json"
	dirPerm        = 0o755
	filePerm       = 0o644
)

// Options configures the filesystem driver.
type Options struct {
	// Indent is the JSON indentation string.
	// Default: "  " (two spaces).
	// CompactJSON, not Indent, selects compact output.
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

// Driver is a filesystem-backed implementation of [store.Driver].
// It stores one JSON file per record and one _schema.json per schema.
type Driver struct {
	root string
	opts Options
	mu   sync.RWMutex
}

// NewDriver creates a new filesystem driver rooted at the given
// directory. The directory is created if it does not exist.
func NewDriver(root string, opts Options) (*Driver, error) {
	opts = opts.withDefaults()

	if err := os.MkdirAll(root, dirPerm); err != nil {
		return nil, fmt.Errorf("[xdb/xdbfs] create root directory: %w", err)
	}

	return &Driver{
		root: root,
		opts: opts,
	}, nil
}

// Root returns the root directory of the driver.
func (d *Driver) Root() string {
	return d.root
}

// Close is a no-op for the filesystem driver.
func (d *Driver) Close() error { return nil }

// Health checks that the root directory exists and is writable.
func (d *Driver) Health(_ context.Context) error {
	info, err := os.Stat(d.root)
	if err != nil {
		return fmt.Errorf("[xdb/xdbfs] root directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("[xdb/xdbfs] root is not a directory: %s", d.root)
	}

	// Check write access by creating and removing a temp file.
	f, err := os.CreateTemp(d.root, ".health-*")
	if err != nil {
		return fmt.Errorf("[xdb/xdbfs] root not writable: %w", err)
	}

	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)

	return nil
}

// writeFileAtomic writes data to path atomically by writing to a temp
// file first, then renaming.
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

// writeFileExclusive writes data to path with O_CREATE|O_EXCL: the
// filesystem guarantees exactly one concurrent creator wins. Returns
// [os.ErrExist] (wrapped) if the file already exists. On a write
// failure the partial file is removed, so a failed create has no
// effect.
func writeFileExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}

	return nil
}

// removeIfEmpty removes dir if it contains no entries.
func removeIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		_ = os.Remove(dir)
	}
}

// isNotExist reports whether err is a missing-file error.
func isNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
