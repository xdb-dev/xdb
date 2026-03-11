// Package xdbredis provides a Redis-backed implementation of [store.Store].
//
// Records are stored as Redis hashes where each tuple attribute maps to
// a hash field. Values are encoded as type-prefixed strings for lossless
// round-tripping without requiring a schema lookup on read.
//
// Schemas are stored as JSON strings using the standard [schema.Def]
// marshal/unmarshal.
//
// The store maintains Redis Set indexes for efficient listing:
//
//   - Record index:    {prefix}:{ns}:{schema}:_idx   — record IDs per schema
//   - Schema index:    {prefix}:{ns}:_idx             — schema names per namespace
//   - Namespace index: {prefix}:_idx                  — all namespace names
//
// Key layout:
//
//	{prefix}:{ns}:{schema}:{id}        → Hash (record data)
//	{prefix}:{ns}:{schema}:_schema     → String (schema JSON)
//	{prefix}:{ns}:{schema}:_idx        → Set (record IDs)
//	{prefix}:{ns}:_idx                 → Set (schema names)
//	{prefix}:_idx                      → Set (namespace names)
package xdbredis
