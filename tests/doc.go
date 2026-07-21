// Package tests provides the shared conformance suites every XDB
// backend runs against. The suites mirror the two layers of the
// storage stack, and a backend proves itself by registering both.
//
// # Two tiers
//
// Driver suites exercise a raw [store.Driver] — pure storage, no
// facade, no enforcement middleware. They pin the storage contract
// itself: the four-op mutation table, tuple read/scan semantics,
// OpCreate atomicity under a race, verbatim Def CRUD, and the optional
// capabilities. A driver suite must not assume any policy (validation,
// revision stamping, namespace derivation) — that lives in the facade.
//
//   - [DriverSuite]  — the core [store.Driver] contract.
//   - [QuerySuite]   — the optional [store.QueryDriver] pushdown
//     capability; skips when unimplemented.
//
// Store suites exercise a full [store.Store] built with
// [store.New](driver), so they run through the facade + enforcement
// middleware + driver stack. They split by what they actually verify.
//
// Per-backend store suites verify facade↔driver integration — that the
// driver faithfully persists and returns what the middleware needs, on
// every backend:
//
//   - [RecordStoreSuite]    — record CRUD, filters, pagination.
//   - [SchemaStoreSuite]    — schema CRUD, revision CAS, array elem.
//   - [NamespaceStoreSuite] — namespaces derived from schemas.
//   - [TupleStoreSuite]     — attr-level (#attr) verbs and policy.
//   - [TypesStoreSuite]     — value fidelity: scalars, typed arrays,
//     object arrays (fidelity is schema-backed, hence a store suite).
//   - [BatchSuite]          — transactions; register only when the
//     driver is a [store.TxDriver] (memory, sqlite).
//   - [BenchmarkSuite]      — throughput benchmarks.
//
// Policy store suites verify enforcement *decisions* — mode rules,
// Required, dynamic evolution, cascade — which live once in the facade
// middleware ([store] enforce) and are driver-independent. Running them
// on every backend would re-test identical code, so they run once
// against the memory reference driver; the driver-facing behaviors they
// rely on (tuple/def storage, DropRecords, PutSchema write-back)
// are pinned per-backend by [DriverSuite] and [TypesStoreSuite]:
//
//   - [ModeStoreSuite]      — flexible/strict/dynamic + object arrays.
//   - [CascadeStoreSuite]   — DeleteSchemaRecords cascade.
//
// # Registering a backend
//
// A new backend adds a driver_test.go running the driver suites
// against its raw driver, and a store_test.go running the store suites
// against store.New(NewDriver(...)):
//
//	func TestDriverSuite(t *testing.T) {
//		tests.NewDriverSuite(func() store.Driver {
//			return mybackend.NewDriver(...)
//		}).Run(t)
//	}
//	func TestQuerySuite(t *testing.T) {
//		tests.NewQuerySuite(func() store.Driver {
//			return mybackend.NewDriver(...)
//		}).Run(t)
//	}
//	func TestTypes(t *testing.T) {
//		tests.NewTypesStoreSuite(func() store.Store {
//			return store.New(mybackend.NewDriver(...))
//		}).Run(t)
//	}
//
// Passing both tiers means the backend behaves identically to every
// other backend. The factory is called before each test group, so
// every group gets a fresh, isolated store. A new backend registers
// the per-backend suites above; the policy suites ([ModeStoreSuite],
// [CascadeStoreSuite]) are registered once, by the memory reference.
//
// [RoundTrip] in rtharness.go is unrelated to the store suites: it is
// the conformance harness for the schema importers (xdbstruct,
// protoimport, jsonschemaimport), which it runs against an in-memory
// store.
package tests
