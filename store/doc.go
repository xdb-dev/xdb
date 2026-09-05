// Package store turns a storage [Driver] into a validated [Store].
//
// A Driver is pure storage. It reads and writes tuples, and it stores
// schema definitions verbatim. It does no validation, no mode
// enforcement, and no revision CAS, and it never sees a [core.Record].
// The drivers xdbmemory, xdbfs, xdbredis, and xdbsqlite implement only
// Driver.
//
// [New] is the only way to get a Store. It wraps a Driver in a facade.
// The facade compiles record, tuple, and schema verbs into driver
// [Mutation]s, assembles records from tuple reads, and derives
// namespaces from schema scans. New always installs the versioning and
// schema-enforcement middleware, so a Store that skips validation
// cannot exist:
//
//	db := store.New(xdbmemory.NewDriver())
//
// # Writes
//
// Every record and tuple verb becomes a [Mutation] that carries an
// [Op]: patch, create, put, or delete. The exists-semantics of a write
// are data that the driver receives, not code that each driver writes
// again. Drivers apply one mutation at a time and return bare sentinel
// errors. The facade sequences batches and attributes the first
// failure to its mutation with [MutationError].
//
// # Middleware stack
//
// The facade builds this stack over the raw driver:
//
//	logging(enforce(cache(versioned(raw))))
//
// The versioning layer maintains three system attributes on every
// record:
//
//   - _id is projected from the record path on every read. It is never
//     stored.
//   - _version is a counter that starts at 1 and increases by one on
//     every write.
//   - _updated is the timestamp of the last write.
//
// A write can carry _version as a compare-and-swap precondition. If the
// value differs from the stored version, the write fails with
// [core.ErrConflict]. A write without _version, or with 0, is
// unconditional.
//
// The enforcement layer holds all schema policy: per-tuple type
// checks, mode rules, dynamic evolution, required-field checks, and
// revision CAS on schema writes. It sits above versioning, so it
// validates the tuples that the caller wrote. The stamped system
// tuples are then stored like any other declared field, and a driver
// knows nothing about versioning.
//
// Options add observability and speed around enforcement. [WithLogger]
// logs writes. [WithSchemaCache] caches schema reads.
//
// # Driver capabilities
//
// The optional capabilities [TxDriver] (native transactions) and
// [QueryDriver] (filter pushdown) are detected once, on the raw
// driver, when New runs. [Closer] and [HealthChecker] are type-asserted
// on the raw driver on each Close and Health call. When the driver
// supports transactions, the returned Store also implements [TX], and
// every write runs in a transaction.
//
// The service layer is the only consumer of Store. RPC handlers never
// touch a Driver directly. Driver implementations must be safe for
// concurrent use.
package store
