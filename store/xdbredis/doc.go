// Package xdbredis provides a Redis-backed implementation of
// [store.Driver]. It is pure storage: no validation, no mode
// enforcement, and no revision logic. That policy lives in the store
// facade. Construct a usable store with
// store.New(xdbredis.NewDriver(client)).
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
// Each mutation is atomic. Patch and delete are single Redis commands.
// The conditional writes (create, put) run as Lua scripts, so their
// existence gate and their write happen in one round trip. The driver
// has no native transactions and deliberately does not implement
// [store.TxDriver]. The facade falls back to sequential writes.
package xdbredis
