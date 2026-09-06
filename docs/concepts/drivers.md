---
title: Drivers
description: The pure-storage contract that drivers implement. Tuple reads, mutations with intent, and verbatim schema CRUD.
package: store, store/xdbmemory, store/xdbfs, store/xdbredis, store/xdbsqlite
---

# Drivers

A **Driver** is the contract that a driver package implements over its backend. The driver boundary speaks exactly one unit: [tuples](tuples.md). Reads return tuples. Writes are batches of per-record mutations that carry intent as data. `core.Record` never crosses this boundary. The [store facade](stores.md) assembles records from tuple reads, and compiles records into mutations on writes.

Drivers are **pure storage**. They do no validation, no mode enforcement, no revision stamping, no namespace derivation, and no versioning. All policy lives in the middleware that `store.New` installs. A new driver implements `Driver` and inherits every guarantee. It cannot forget validation, because validation was never its job.

This includes [per-record versioning](versioning.md). `_version` and `_updated` reach a driver as ordinary tuples of ordinary declared fields, in the same mutation as the own data of the record. A driver stores them like any other tuple, and needs to know nothing about them.

`Driver` composes four storage roles. Each role is a small interface that a wrapper can depend on in isolation:

```go
type TupleReader interface {
    GetTuples(ctx context.Context, uris ...*core.URI) ([]*core.Tuple, error)
    ScanTuples(ctx context.Context, scope *core.URI) iter.Seq2[*core.Tuple, error]
}

type TupleWriter interface {
    Apply(ctx context.Context, m store.Mutation) error   // one mutation, intent as data
}

// Definitions are stored verbatim.
type SchemaReader interface {
    GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error)
    ScanSchemas(ctx context.Context, scope *core.URI) iter.Seq2[*schema.Def, error]
}

type SchemaWriter interface {
    CreateSchema(ctx context.Context, def *schema.Def) error   // atomic exists-fail
    PutSchema(ctx context.Context, def *schema.Def) error      // unconditional upsert
    DeleteSchema(ctx context.Context, uri *core.URI) error
    DropRecords(ctx context.Context, uri *core.URI) error      // record-space cleanup
}

type Driver interface {
    TupleReader
    TupleWriter
    SchemaReader
    SchemaWriter
}
```

A `Driver` is deliberately not a `Store`. Code that uses a driver directly (for example, driver tests) is visibly unenforced.

## Reading Tuples

- `GetTuples` is a batch point read by attribute-level URIs. Absent attributes are **omitted**, and the result keeps request order. A batch read does not fail on absence. The facade maps absence to `ErrNotFound` for singular reads.
- `ScanTuples` yields every tuple under a scope: a namespace, a schema, or a record path. A record read is a scan of one path. The tuples of one record are yielded **contiguously**, so the facade assembles records in a single pass, without a buffer across records.

## Writing: Four Ops, One Table

Writes arrive one mutation at a time. Exists-semantics are data that the driver receives, not code that it invents:

```go
type Mutation struct {
    Path   *core.URI     // record path (ns + schema + id)
    Tuples []*core.Tuple // puts, for OpPatch/OpCreate/OpPut
    Attrs  []string      // removals, for OpDelete (empty = whole record)
    Op     Op
}
```

| Op | Path absent | Path exists | Note |
|----|-------------|-------------|------|
| `OpPatch` | creates record | adds or overwrites the named attributes | everything else at the path is untouched (HTTP PATCH) |
| `OpCreate` | writes full set | `ErrAlreadyExists` | **must be atomic**. This is the one op where check-then-write loses data (HTTP POST) |
| `OpPut` | writes full set | replaces full set | attributes not in the set are removed. Upsert (HTTP PUT) |
| `OpDelete` | no-op | removes the named attributes, or the whole record | idempotent (HTTP DELETE) |

`Apply` executes one mutation atomically and returns bare sentinel errors (`core.ErrAlreadyExists` for a create on an existing path). Batch sequencing and error attribution live in the facade. The facade feeds mutations one at a time, stops at the first failure, and wraps it in a `*store.MutationError{Index, Path, Err}`. Mutations before the failure remain applied, and the failing mutation has no partial effect. Whole-batch atomicity is also the job of the facade, which orchestrates it on a `TxDriver`.

A record is its tuple set. A write of an empty set stores nothing, and the removal of the last tuple of a record removes the record. An existence check always asks one question: does the path hold any tuples?

Some drivers rebuild the full tuple set of a record to apply a mutation, for example the file-per-record `xdbfs` and the row-per-record `xdbsqlite`. These drivers can reuse `store.MergeTuples` to fold an `OpPatch` onto the current tuples, and `store.RemoveAttrs` to remove attributes for an `OpDelete`.

## Schema Definitions, Verbatim

Drivers store definitions **verbatim**. The revision arrives already stamped, validation already ran, and the CAS already passed. The create/put split (`CreateSchema`/`PutSchema`) keeps exists-semantics data-driven, and mirrors `OpCreate`/`OpPut`. `DropRecords` is the record-space cleanup (DROP TABLE, key sweep, file removal). The definition itself stays. Driver and store share the `Schema` verb names deliberately. The difference is the layer, not the name: `Store.GetSchema` is validated policy over the raw, verbatim storage of `Driver.GetSchema`.

## Optional Capabilities

The facade detects `TxDriver` and `QueryDriver` once, on the raw driver, at `store.New` time. It looks for `store.Closer` and `store.HealthChecker` on the raw driver when `Close` or `Health` is called. Middleware wrappers never hide a capability:

```go
// Native transactions. fn receives a tx-scoped Driver; error = rollback.
type TxDriver interface {
    Tx(ctx context.Context, fn func(tx store.Driver) error) error
}

// Native filter pushdown: one page item per matching record (its full
// tuple set). Return store.ErrUnsupportedQuery to make the facade
// synthesize the list from a scan instead.
type QueryDriver interface {
    QueryTuples(ctx context.Context, q *store.Query) (*store.Page[[]*core.Tuple], error)
}
```

A driver can also implement `store.Closer` to release resources, and `store.HealthChecker` to report connectivity.

## How Each Driver Maps the Contract

How a mutation lands on storage is the internal concern of the driver. The interfaces promise semantics, not strategy.

| | `xdbmemory` | `xdbfs` | `xdbredis` | `xdbsqlite` |
|---|---|---|---|---|
| Record storage | `map[path]map[attr]*Tuple` | one JSON file per record | one hash per record | column table (strict/dynamic) or KV table (flexible/schema-free) per schema |
| `OpCreate` atomicity | map check under lock | `O_CREATE\|O_EXCL` | Lua EXISTS-gated write | exists check under write mutex + SQL tx |
| Def storage | map | `_schema.json` | JSON at `…:_schema` key | `_schemas` table (JSON), plus DDL for column tables |
| `TxDriver` | yes (snapshot + rollback) | no | no | yes (`sql.Tx`) |
| `QueryDriver` | no | no | no | yes (CEL → SQL via `filter/sqlgen`) |

Notes:

- **sqlite** routes a mutation to one of two engines: a column-table engine (strict/dynamic) or a KV engine (flexible/schema-free). The stored definition selects the engine at a single point (`engineFor`). The op semantics run once, above the engines. This is storage strategy, not policy. KV rows store values in their native SQLite storage class, so filter comparisons stay numeric. `PutSchema` diffs the field sets and issues `ALTER TABLE` for added and removed columns. Type changes never reach the driver: `schema.ValidateUpdate` rejects them in middleware, as it rejects mode changes.
- **indexed/unique fields** materialize only in the column engine of sqlite. `ensure` and `evolve` issue `CREATE [UNIQUE] INDEX` per flagged field, and remove the index before the column on removal. A unique violation surfaces as `core.ErrUniqueViolation`. Other drivers store the markers verbatim and build no index. Like all schema metadata, the marker is data that the driver can ignore. The store adds no portable enforcement above them: `indexed` and `unique` are backend capabilities, so a backend without an index accepts a duplicate.
- **redis** ships without `TxDriver` deliberately. The facade uses its sequential fallback there.

## The Conformance Suite

`tests.NewDriverSuite` pins the contract. It covers the four-op table with per-mutation semantics and bare sentinel errors. It also covers absence semantics, scan contiguity, `OpCreate` under a 16-goroutine race, and verbatim definition CRUD:

```go
func TestDriverSuite(t *testing.T) {
    tests.NewDriverSuite(func() store.Driver {
        return mybackend.NewDriver(...)
    }).Run(t)
}
```

A new driver registers the driver suite and the query suite (`tests.NewQuerySuite`, which skips without pushdown) against its raw driver. It registers the store suites (record, schema, namespace, tuple, types, version) against `store.New(NewDriver(...))`. A driver with native transactions also registers the batch suite. The mode and cascade suites test facade policy, so they run once, on the memory driver. A driver that passes both tiers behaves identically to every other driver.

## Related Concepts

- [Stores](stores.md) — The facade and middleware above drivers
- [Tuples](tuples.md) — The unit the driver boundary speaks
- [Schemas](schemas.md) — Definitions that drivers store verbatim
