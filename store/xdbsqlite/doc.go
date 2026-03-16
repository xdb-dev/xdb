// Package xdbsqlite provides a SQLite-backed implementation of [store.Store].
//
// The store uses a dual storage strategy based on schema mode:
//
//   - Flexible schemas (and schema-less records) use per-schema KV tables
//     where each row stores one attribute of one record.
//   - Strict and dynamic schemas use dedicated SQL tables with columns
//     derived from the schema field definitions.
//
// Schema definitions are stored in a bootstrap _schemas table as JSON.
// Namespaces are derived from registered schemas.
//
// Table naming:
//
//	_schemas                     → schema metadata (bootstrap)
//	"kv:<ns>/<schema>"           → KV table for flexible/schema-less records
//	"t:<ns>/<schema>"            → column table for strict/dynamic schemas
package xdbsqlite
