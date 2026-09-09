// Package store turns a storage [Driver] into a validated [Store].
//
// A Driver reads and writes tuples and stores schema definitions verbatim.
// [New] wraps it in a facade with versioning and schema enforcement:
//
//	db := store.New(xdbmemory.NewDriver())
//
// The facade assembles [core.Record] values from tuple reads and derives
// namespaces from schema scans. Record and tuple writes become [Mutation]s;
// schema writes use the driver's schema methods.
//
// # Writes
//
// Each Mutation carries an [Op] that specifies patch, create, put, or delete
// semantics. Drivers apply one mutation atomically. The facade sequences
// batches and reports the first failure with [MutationError]. A driver that
// implements [TxDriver] also supports rollback of the whole batch.
//
// # Middleware stack
//
// The facade builds this stack over the raw driver. Logging and caching
// are optional:
//
//	logging(enforce(cache(versioned(raw))))
//
// Enforcement checks caller-supplied tuples against the schema before
// versioning adds the stored system attributes. It also handles dynamic
// schema evolution, required fields, and schema revision checks.
//
// Records expose these system attributes:
//
//   - _id is projected from the record path by the facade on reads.
//   - _version starts at 1 and increments when a write stamps the record.
//     Empty patches do not increment it. Deleting a record removes it.
//   - _updated is the timestamp of the last write that stamped the record.
//
// A non-zero _version on a record write is checked against the stored
// version. A mismatch returns [core.ErrConflict]. An omitted or zero
// _version requests an unconditional write. Whole-record delete
// preconditions are checked by the service layer.
//
// Version checks and writes run in one transaction on [TxDriver] backends.
// On filesystem and Redis backends they are separate operations, so
// concurrent writes can both pass the same version check. Schema revision
// checks have the same limitation.
//
// [WithLogger] logs writes. [WithSchemaCache] caches schema reads; transactions
// bypass the cache so they can read their own schema changes. A successful
// transaction invalidates the cache.
//
// # Driver capabilities
//
// [New] detects [TxDriver] and [QueryDriver] on the raw driver. A Store backed
// by a TxDriver also implements [TX], and its writes run in transactions.
// QueryDriver supports native filter pushdown; other drivers use tuple scans
// and in-memory filtering.
//
// [Closer] and [HealthChecker] are checked on the raw driver when Close or
// Health is called. Driver implementations must be safe for concurrent use.
package store
