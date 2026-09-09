---
title: Drivers
description: The pure-storage contract that drivers implement. Tuple reads, mutations with intent, and verbatim schema CRUD.
package: store, store/xdbmemory, store/xdbfs, store/xdbredis, store/xdbsqlite
---

# Drivers

A `Driver` reads [tuples](tuples.md), applies record mutations, and stores schema definitions on a backend. The [store facade](stores.md) assembles records from tuple reads and converts records into mutations on writes. `core.Record` stays in the facade.

The middleware installed by `store.New` handles validation, schema modes, revisions, and record versions. The facade derives namespaces. Drivers implement storage operations and receive data that has passed these checks.

The middleware writes [record versions](versioning.md) as `_version` and `_updated` tuples in the same mutation as the user data. Drivers store them as ordinary fields.

`Driver` combines interfaces for tuple and schema reads and writes. A wrapper can depend on an individual interface:

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

Direct driver calls, including calls in driver tests, bypass store validation and versioning.

## Naming

Verbs that retrieve a value are `Get*`, writes are `Put*`/`Create*`/`Delete*`, and iteration is `Scan*`. This KV-shaped vocabulary is deliberate, chosen for symmetry across `Driver` and `Store`. Go convention would drop the `Get` prefix; XDB keeps it.

The prefix means the method retrieves a value, so a method returning only a yes/no is named for what it asks. That is why the namespace check is `NamespaceExists` rather than `GetNamespace`.

## Reading Tuples

- `GetTuples` is a batch point read by attribute-level URIs. Absent attributes are omitted, and the result keeps request order. A batch read does not fail on absence. The facade maps absence to `ErrNotFound` for singular reads.

- `ScanTuples` yields every tuple under a scope: a namespace, a schema, or a record path. A record read is a scan of one path. The tuples of one record are yielded contiguously, so the facade assembles records in a single pass, without a buffer across records.

## Mutation Behavior

Each mutation specifies how to handle an existing or absent record:

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
| `OpCreate` | writes full set | `ErrAlreadyExists` | must be atomic. This is the one op where check-then-write loses data (HTTP POST) |
| `OpPut` | writes full set | replaces full set | attributes not in the set are removed. Upsert (HTTP PUT) |
| `OpDelete` | no-op | removes the named attributes, or the whole record | idempotent (HTTP DELETE) |

`Apply` executes one mutation atomically and returns bare sentinel errors (`core.ErrAlreadyExists` for a create on an existing path). Batch sequencing and error attribution live in the facade. The facade feeds mutations one at a time, stops at the first failure, and wraps it in a `*store.MutationError{Index, Path, Err}`. Mutations before the failure remain applied, and the failing mutation has no partial effect. Whole-batch atomicity is also the job of the facade, which orchestrates it on a `TxDriver`.

A record is its tuple set. A write of an empty set stores nothing, and the removal of the last tuple of a record removes the record. An existence check always asks one question: does the path hold any tuples?

Some drivers rebuild the full tuple set of a record to apply a mutation, for example the file-per-record `xdbfs` and the row-per-record `xdbsqlite`. These drivers can reuse `store.MergeTuples` to fold an `OpPatch` onto the current tuples, and `store.RemoveAttrs` to remove attributes for an `OpDelete`.

## Schema Definitions, Verbatim

Drivers store definitions verbatim after middleware validates them, checks the revision, and stamps the new revision. `CreateSchema` fails if the definition exists; `PutSchema` replaces or creates it. `DropRecords` removes the schema's records and keeps its definition. `Store.GetSchema` applies store policy over `Driver.GetSchema`.

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

## Backend Storage Layouts

Each driver chooses a storage layout while following the same mutation contract.

| | `xdbmemory` | `xdbfs` | `xdbredis` | `xdbsqlite` |
|---|---|---|---|---|
| Record storage | `map[path]map[attr]*Tuple` | one JSON file per record | one hash per record | column table (strict/dynamic) or KV table (flexible/schema-free) per schema |
| `OpCreate` atomicity | map check under lock | `O_CREATE\|O_EXCL` | Lua EXISTS-gated write | exists check under write mutex + SQL tx |
| Def storage | map | `_schema.json` | JSON at `…:_schema` key | `_schemas` table (JSON), plus DDL for column tables |
| `TxDriver` | yes (snapshot + rollback) | no | no | yes (`sql.Tx`) |
| `QueryDriver` | no | no | no | yes (CEL -> SQL via `filter/sqlgen`) |

Notes:

- sqlite routes a mutation to one of two engines: a column-table engine (strict/dynamic) or a KV engine (flexible/schema-free). The stored definition selects the engine at a single point (`engineFor`). The op semantics run once, above the engines.  KV rows store values in their native SQLite storage class, so filter comparisons stay numeric. `PutSchema` diffs the field sets and issues `ALTER TABLE` for added and removed columns. Type changes never reach the driver: `schema.ValidateUpdate` rejects them in middleware, as it rejects mode changes.

- indexed/unique fields materialize only in the column engine of sqlite. `ensure` and `evolve` issue `CREATE [UNIQUE] INDEX` per flagged field, and remove the index before the column on removal. A unique violation surfaces as `core.ErrUniqueViolation`. Other drivers store the markers verbatim and build no index.  The store adds no portable enforcement above them: `indexed` and `unique` are backend capabilities, so a backend without an index accepts a duplicate.

- Redis does not implement `TxDriver`. The facade uses its sequential fallback there.

## The Conformance Suite

`storetest.NewDriverSuite` pins the contract. It covers the four-op table with per-mutation semantics and bare sentinel errors. It also covers absence semantics, scan contiguity, `OpCreate` under a 16-goroutine race, and verbatim definition CRUD:

```go
func TestDriverSuite(t *testing.T) {
    storetest.NewDriverSuite(func() store.Driver {
        return mybackend.NewDriver(...)
    }).Run(t)
}
```

A new driver registers the driver suite and the query suite (`storetest.NewQuerySuite`, which skips without pushdown) against its raw driver. It registers the store suites (record, schema, namespace, tuple, types, version) against `store.New(NewDriver(...))`. A driver with native transactions also registers the batch suite. The mode and cascade suites test facade policy, so they run once, on the memory driver. These suites check the shared contract and the capabilities each driver supports.

## Related Concepts

- [Stores](stores.md): The facade and middleware above drivers

- [Tuples](tuples.md): The data drivers read and write

- [Schemas](schemas.md): Definitions that drivers store verbatim
