// Package xdbredis provides a Redis-backed implementation of
// [store.Driver]. Use [store.New] to add schema enforcement and versioning:
//
//	st := store.New(xdbredis.NewDriver(client))
//
// Records are stored as Redis hashes: one hash per record, one field
// per attribute. Values are encoded as type-prefixed strings, so they
// round-trip without loss and a read needs no schema lookup. Schema
// definitions are stored verbatim as JSON strings.
//
// Key layout:
//
//	{prefix}:{ns}:{schema}:{id}        → Hash (record attrs)
//	{prefix}:{ns}:{schema}:_schema     → String (schema Def JSON)
//
// Scans discover records with SCAN + MATCH over the key pattern. There
// are no secondary indexes. A record with zero tuples does not exist:
// Redis removes a hash automatically when its last field is deleted,
// and replace-style mutations with no tuples leave no key behind.
//
// Each mutation runs as a single Redis command or Lua script. Create
// checks existence before writing; put deletes the old hash and writes
// its replacement. The driver does not implement [store.TxDriver], so
// the facade applies batches and version checks sequentially.
package xdbredis
