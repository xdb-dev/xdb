---
title: Drivers
description: The pure-storage contract backends implement — tuple reads, mutations with intent, and verbatim schema CRUD.
package: store, store/xdbmemory, store/xdbfs, store/xdbredis, store/xdbsqlite
---

# Drivers

A **Driver** is what a storage backend implements. The driver boundary speaks exactly one unit: [tuples](tuples.md). Reads return tuples; writes are batches of per-record mutations carrying intent as data. `core.Record` never crosses this boundary — records are assembled by the [store facade](stores.md) from tuple reads and compiled into mutations on writes.

Drivers are **pure storage**. No validation, no mode enforcement, no revision stamping, no namespace derivation — all policy lives in the middleware that `store.New` installs. A new backend implements `Driver` and inherits every guarantee; it cannot forget validation, because validation was never its job.

`Driver` composes four storage roles, each a small interface a wrapper can
depend on in isolation:

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

A `Driver` is deliberately not a `Store` — using one directly (e.g. in driver tests) is visibly unenforced.

## Reading Tuples

- `GetTuples` is a batch point read by attr-level URIs. Absent attrs are **omitted**, in request order — batch reads don't error on absence; the facade maps absence to `ErrNotFound` where singular.
- `ScanTuples` yields every tuple under a scope: a namespace, a schema, or a record path. A record read is a scan of one path. Tuples of one record are yielded **contiguously** — the facade assembles records in a single pass without buffering across records.

## Writing: Four Ops, One Table

Writes arrive one mutation at a time. Exists-semantics are data the driver receives, not code it invents:

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
| `OpPatch` | creates record | adds/overwrites named attrs | everything else at the path untouched (HTTP PATCH) |
| `OpCreate` | writes full set | `ErrAlreadyExists` | **must be atomic** — the one op where check-then-write loses data (HTTP POST) |
| `OpPut` | writes full set | replaces full set | absent attrs are dropped, upsert (HTTP PUT) |
| `OpDelete` | no-op | removes named attrs / whole record | idempotent (HTTP DELETE) |

`Apply` executes one mutation atomically and returns bare sentinel errors (`core.ErrAlreadyExists` for a create on an existing path). Batch sequencing and error attribution live in the facade: it feeds mutations one at a time, stops at the first failure, and wraps it in a `*store.MutationError{Index, Path, Err}` — mutations before the failure remain applied, and the failing mutation has no partial effect. Whole-batch atomicity is also the facade's job, orchestrated on `TxDriver`s.

A record is its tuple set: writing an empty set stores nothing, and removing a record's last tuple removes the record. Existence checks are always "does the path hold any tuples".

Backends that reconstruct a record's full tuple set to apply a mutation (file- or row-per-record stores like `xdbfs` and `xdbsqlite`) can reuse `store.MergeTuples` to fold an `OpPatch` onto the current tuples and `store.RemoveAttrs` to drop attrs for an `OpDelete`.

## Schema Definitions, Verbatim

Drivers store definitions **verbatim** — the revision arrives already stamped, validation already ran, the CAS already passed. The create/put split (`CreateSchema`/`PutSchema`) exists so exists-semantics stay data-driven, mirroring `OpCreate`/`OpPut`. `DropRecords` is the record-space cleanup (DROP TABLE, key sweep, file removal); the def itself stays. Driver and store share the `Schema` verbs deliberately — the distinction is the layer, not the name: `Store.GetSchema` is validated policy over `Driver.GetSchema`'s raw, verbatim storage.

## Optional Capabilities

Capabilities are detected once, on the raw driver, at `store.New` time — middleware wrappers never hide them:

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

Plus `store.Closer` and `store.HealthChecker`, as before.

## How Each Backend Maps the Contract

How a mutation lands on storage is the driver's internal concern; the interfaces promise semantics, not strategy.

| | `xdbmemory` | `xdbfs` | `xdbredis` | `xdbsqlite` |
|---|---|---|---|---|
| Record storage | `map[path]map[attr]*Tuple` | one JSON file per record | one hash per record | column table (strict/dynamic) or KV table (flexible/schema-less) per schema |
| `OpCreate` atomicity | map check under lock | `O_CREATE\|O_EXCL` | Lua EXISTS-gated write | exists check under write mutex + SQL tx |
| Def storage | map | `_schema.json` | JSON at `…:_schema` key | `_schemas` table (JSON), plus DDL for column tables |
| `TxDriver` | yes (snapshot + rollback) | no | no | yes (`sql.Tx`) |
| `QueryDriver` | no | no | no | yes (CEL → SQL via `filter/sqlgen`) |

Notes:

- **sqlite** routes a mutation to one of two engines — a column-table engine (strict/dynamic) or a KV engine (flexible/schema-less) — chosen from the stored def at a single point (`engineFor`); the op semantics run once above the engines. This is storage strategy, not policy. KV rows store values in their native SQLite storage class so filter comparisons stay numeric. `PutSchema` diffs field sets and issues `ALTER TABLE` for added/removed columns (type changes never reach the driver; `schema.ValidateUpdate` rejects them in middleware, as it does mode changes).
- **redis** ships without `TxDriver` deliberately; the facade uses its sequential fallback there.

## The Conformance Suite

`tests.NewDriverSuite` pins the contract — the four-op table with per-mutation semantics and bare sentinel errors, absence semantics, scan contiguity, `OpCreate` under a 16-goroutine race, and verbatim def CRUD:

```go
func TestDriverSuite(t *testing.T) {
    tests.NewDriverSuite(func() store.Driver {
        return mybackend.NewDriver(...)
    }).Run(t)
}
```

A new backend registers the driver suite against its raw driver, and the store suites (records, schemas, namespaces, modes, cascade, tuples) against `store.New(NewDriver(...))`. Passing both means it behaves identically to every other backend.

## Related Concepts

- [Stores](stores.md) — The facade and middleware above drivers
- [Tuples](tuples.md) — The unit the driver boundary speaks
- [Schemas](schemas.md) — Definitions drivers store verbatim
