package xdbfs

import (
	"path/filepath"

	"github.com/xdb-dev/xdb/core"
)

// recordPath returns the file path for a record.
// e.g., root/myapp/users/user-1.json.
func (s *Store) recordPath(uri *core.URI) string {
	return filepath.Join(
		s.root,
		uri.NS().String(),
		uri.Schema().String(),
		uri.ID().String()+jsonExt,
	)
}

// schemaPath returns the file path for a schema definition.
// e.g., root/myapp/users/_schema.json.
func (s *Store) schemaPath(uri *core.URI) string {
	return filepath.Join(
		s.root,
		uri.NS().String(),
		uri.Schema().String(),
		schemaFileName,
	)
}

// nsDir returns the directory for a namespace.
// e.g., root/myapp/.
func (s *Store) nsDir(uri *core.URI) string {
	return filepath.Join(s.root, uri.NS().String())
}

// schemaDir returns the directory for a schema.
// e.g., root/myapp/users/.
func (s *Store) schemaDir(uri *core.URI) string {
	return filepath.Join(
		s.root,
		uri.NS().String(),
		uri.Schema().String(),
	)
}
