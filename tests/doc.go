// Package tests provides the shared conformance suites that every XDB
// driver runs. The suites mirror the two layers of the storage stack. A
// driver proves itself when it registers both layers.
//
// # Two tiers
//
// Driver suites exercise a raw [store.Driver]: pure storage, no facade, no
// enforcement middleware. They pin the storage contract itself: the four-op
// mutation table, tuple read and scan semantics, OpCreate atomicity under a
// race, verbatim Def CRUD, and the optional capabilities. A driver suite
// must not assume any policy (validation, revision stamping, namespace
// derivation). Policy lives in the facade.
//
//   - [DriverSuite]  — the core [store.Driver] contract.
//   - [QuerySuite]   — the optional [store.QueryDriver] pushdown
//     capability. The suite skips when the driver does not implement it.
//
// Store suites exercise a full [store.Store] that [store.New] builds from a
// driver. As a result, they run through the whole stack: facade, middleware,
// and driver. They split by what they verify.
//
// Per-driver store suites verify the integration of the facade and the
// driver. They make sure that the driver persists and returns what the
// middleware needs, on every backend:
//
//   - [RecordStoreSuite]    — record CRUD, filters, pagination.
//   - [SchemaStoreSuite]    — schema CRUD, revision CAS, array elem.
//   - [NamespaceStoreSuite] — namespaces derived from schemas.
//   - [TupleStoreSuite]     — attribute-level (#attr) verbs and policy.
//   - [TypesStoreSuite]     — value fidelity: scalars, typed arrays,
//     object arrays. Fidelity is schema-backed, so this is a store suite.
//   - [VersionSuite]        — per-record versioning: _version and
//     _updated stamping, compare-and-swap, lifecycle, filtering.
//   - [BatchSuite]          — transactions. Register it only when the
//     driver is a [store.TxDriver] (memory, sqlite).
//   - [BenchmarkSuite]      — throughput benchmarks.
//
// Policy store suites verify enforcement decisions: mode rules, Required,
// dynamic evolution, and cascade. These decisions live once in the facade
// middleware ([store] enforce) and do not depend on the driver. A run on
// every backend tests identical code again, so they run once, against
// the memory reference driver. The driver-facing behaviors they rely on
// (tuple and def storage, DropRecords, PutSchema write-back) are pinned per
// driver by [DriverSuite] and [TypesStoreSuite]:
//
//   - [ModeStoreSuite]      — flexible/strict/dynamic + object arrays.
//   - [CascadeStoreSuite]   — DeleteSchemaRecords cascade.
//
// # Registering a driver
//
// A new driver adds a driver_test.go that runs the driver suites against
// its raw driver. It also adds a store test file that runs the store suites
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
// When both tiers pass, the driver behaves identically to every other
// driver. The factory is called before each test group, so every group gets
// a fresh, isolated store. A new driver registers the per-driver suites
// above. The policy suites ([ModeStoreSuite], [CascadeStoreSuite]) are
// registered once, by the memory reference driver.
//
// [RoundTrip] in rtharness.go is unrelated to the store suites. It is the
// conformance harness for the schema importers (xdbstruct, xdbproto,
// xdbjson). It runs them against an in-memory store.
package tests
