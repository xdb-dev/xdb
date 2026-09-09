// Package storetest provides the shared conformance suites that pin XDB
// driver and store behavior across every backend.
//
// # Driver suites
//
// [DriverSuite] tests raw [store.Driver] operations, including mutation
// semantics, concurrent creates, tuple scans, and verbatim schema storage.
// [QuerySuite] tests native filter pushdown when available and filter
// behavior through the facade on every backend.
//
// # Store suites
//
// Store suites test a driver wrapped with [store.New]. Register these suites
// for each backend:
//
//   - [RecordStoreSuite]: record CRUD, filters, and pagination.
//   - [SchemaStoreSuite]: schema CRUD and revision checks.
//   - [NamespaceStoreSuite]: namespace discovery from schemas.
//   - [TupleStoreSuite]: attribute reads, patches, and deletes.
//   - [TypesStoreSuite]: scalar and array storage fidelity.
//   - [VersionSuite]: system metadata and sequential version checks.
//   - [BatchSuite]: transactions; requires [store.TxDriver].
//   - [BenchmarkSuite]: throughput benchmarks.
//
// [ModeStoreSuite] and [CascadeStoreSuite] run against the memory driver to
// check schema enforcement and cascade behavior. DriverSuite checks the
// storage operations they depend on for each backend.
//
// # Registering a driver
//
// Run driver suites against a fresh raw driver and store suites against
// store.New(NewDriver(...)):
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
// Each factory call must return an isolated driver or store. Suite
// constructors document when they call the factory.
//
// # Importer tests
//
// [RunRoundTrip] checks schema import and typed data conversion against an
// in-memory store. Each [RoundTrip] supplies the imported schema and the
// source format's encoding and decoding functions.
package storetest
