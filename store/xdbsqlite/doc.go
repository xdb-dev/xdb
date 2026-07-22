// Package xdbsqlite provides a SQLite-backed implementation of
// [store.Driver], with native transactions ([store.TxDriver]) and CEL
// filter pushdown ([store.QueryDriver]). It is pure storage — no
// validation, mode enforcement, or revision logic; that policy lives in
// the store facade. Construct a usable store with
// store.New(xdbsqlite.NewDriver(db)).
//
// # Architecture
//
// A routing Driver owns the database, the write mutex, and transaction
// lifecycle; a [session] carries the store.Driver logic over one query
// handle (the database for lock-free reads, a transaction for writes).
// Each schema's records are stored by one of two engines, chosen by the
// single routing point engineFor from the schema's stored def:
//
//   - The KV engine backs flexible schemas and schema-less records: a
//     per-schema table with one row per attribute.
//   - The table engine backs strict and dynamic schemas: a per-schema
//     table with one column per field.
//
// The engines sit over the raw statement layer in internal/sql; the
// four mutation ops (patch/create/put/delete) execute once in
// runMutation over the engine interface, so both layouts share their
// semantics.
//
// # Storage
//
// Schema definitions are stored in a bootstrap _schemas table as JSON.
// Namespaces are derived from registered schemas. Backing tables are
// created when a schema is created and, lazily, on the first write to a
// schema-less table or after DropRecords.
//
// KV rows store each value in its native SQLite storage class (an ANY
// column), with _type and _elem recording the [core.Type] so values
// decode and SQL comparisons stay numeric. A per-table (_attr,_val)
// index serves filter pushdown.
//
// Table naming:
//
//	_schemas               → schema metadata (bootstrap)
//	"kv:<ns>/<schema>"      → KV table for flexible/schema-less records
//	"t:<ns>/<schema>"       → column table for strict/dynamic schemas
//	"ix:kv:<ns>/<schema>"   → (_attr,_val) index on the KV table
//
// The on-disk format is not compatible with earlier versions of this
// package; there is no migration.
package xdbsqlite
