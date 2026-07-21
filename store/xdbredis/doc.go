// Package xdbredis provides a Redis-backed implementation of
// [store.Driver]: pure storage with no validation, mode enforcement,
// or revision logic — policy lives in the store facade. Construct a
// usable store with store.New(xdbredis.NewDriver(client)).
//
// Records are stored as Redis hashes — one hash per record, one field
// per attribute. Values are encoded as type-prefixed strings for
// lossless round-tripping without requiring a schema lookup on read.
// Schema definitions are stored verbatim as JSON strings.
//
// Key layout:
//
//	{prefix}:{ns}:{schema}:{id}        → Hash (record attrs)
//	{prefix}:{ns}:{schema}:_schema     → String (schema Def JSON)
//
// Scans discover records with SCAN + MATCH over the key pattern; there
// are no secondary indexes. A record with zero tuples does not exist:
// Redis removes a hash automatically when its last field is deleted,
// and replace-style mutations with no tuples leave no key behind.
//
// Mutations are atomic per mutation: patch and delete are single Redis
// commands, while the conditional writes (create, put) run as Lua
// scripts so their existence gate and write happen in one round trip.
// The driver has no native transactions and deliberately does not
// implement [store.TxDriver]; the facade falls back to sequential
// writes.
package xdbredis
