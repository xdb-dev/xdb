// Package store turns a storage [Driver] into a validated [Store].
//
// A Driver is pure storage: it reads and writes tuples and stores schema
// definitions verbatim. It performs no validation, no mode enforcement, and no
// revision CAS, and it never sees a [core.Record]. Backends — xdbmemory,
// xdbfs, xdbredis, xdbsqlite — implement only Driver.
//
// [New] is the only way to obtain a Store. It wraps a Driver in a facade that
// compiles record, tuple, and schema verbs into driver [Mutation]s, assembles
// records from tuple reads, and derives namespaces from schema scans. New
// always installs the schema-enforcement middleware, so a Store that skips
// validation cannot be constructed:
//
//	db := store.New(xdbmemory.NewDriver())
//
// Writes flow through one choke point. Every record and tuple verb becomes a
// [Mutation] carrying an [Op] — patch, create, put, or delete — so
// exists-semantics are data the driver receives, not code each backend
// reinvents. Drivers apply one mutation at a time and return bare sentinel
// errors; the facade sequences batches and attributes the first failure via
// [MutationError]. Enforcement (per-tuple type checks, mode rules, dynamic
// evolution, required-field checks, and revision CAS on schema writes) lives
// in that one middleware layer, above every backend.
//
// Options add observability and acceleration around enforcement: [WithLogger]
// logs writes, [WithSchemaCache] caches schema reads. Optional driver
// capabilities — [TxDriver] for native transactions, [QueryDriver] for filter
// pushdown — are detected once, on the raw driver, at New time. When the driver
// supports transactions the returned Store also implements [TX] and runs every
// write in a transaction.
//
// The service layer is the only consumer of Store — RPC handlers never touch a
// Driver directly. Driver implementations must be safe for concurrent use.
package store
