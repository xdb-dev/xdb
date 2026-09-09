// Package xdbsqlite provides a SQLite-backed implementation of
// [store.Driver], with native transactions ([store.TxDriver]) and CEL
// filter pushdown ([store.QueryDriver]). Use [store.New] to add schema
// enforcement and versioning:
//
//	st := store.New(xdbsqlite.NewDriver(db))
//
// # Architecture
//
// A routing Driver owns the database, the write mutex, and the
// transaction lifecycle. A session carries the store.Driver logic over
// one query handle: the database for lock-free reads, or a transaction
// for writes. One of two engines stores the records of each schema.
// The single routing point engineFor selects the engine from the stored
// definition of the schema:
//
//   - The KV engine backs flexible schemas and schema-free records: a
//     per-schema table with one row per attribute.
//   - The table engine backs strict and dynamic schemas: a per-schema
//     table with one column per field.
//
// The engines sit over the raw statement layer in internal/sql. The
// four mutation ops (patch, create, put, delete) run once in
// runMutation over the engine interface, so both layouts share their
// semantics.
//
// # Storage
//
// Schema definitions are stored as JSON in a bootstrap _schemas table.
// Namespaces are derived from registered schemas. A backing table is
// created when a schema is created. It is also created lazily, on the
// first write to a schema-free table or after DropRecords.
//
// KV rows store each value in its native SQLite storage class (an ANY
// column). The _type and _elem columns record the [core.Type], so
// values decode correctly and SQL comparisons stay numeric. A
// per-table (_attr,_val) index serves filter pushdown. On a column
// table, each indexed or unique field gets its own index.
//
// Table naming:
//
//	_schemas                     → schema metadata (bootstrap)
//	"kv:<ns>/<schema>"           → KV table for flexible/schema-free records
//	"t:<ns>/<schema>"            → column table for strict/dynamic schemas
//	"ix:kv:<ns>/<schema>"        → (_attr,_val) index on the KV table
//	"ix:t:<ns>/<schema>:<field>" → index for an indexed or unique field on the column table
//
// The on-disk format is not compatible with earlier versions of this
// package. There is no migration.
package xdbsqlite
