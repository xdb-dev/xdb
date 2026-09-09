---
title: Stores
description: The store facade, the enforcement and versioning middleware, and the drivers that persist records, schemas, and namespaces on every backend.
package: store, store/xdbmemory, store/xdbfs, store/xdbredis, store/xdbsqlite
---

# Stores

A `Store` reads and writes [records](records.md), [tuples](tuples.md), and [schemas](schemas.md), and lists [namespaces](namespaces.md). It applies shared validation and versioning rules across backends. Transaction and index support depend on the driver.

The store separates policy from persistence:

- The facade and middleware (`store` package) own all policy: schema validation, mode enforcement, dynamic evolution, revision CAS, and record versioning. They also own record assembly, filtering, pagination, namespace derivation, and transaction orchestration.

- [Drivers](drivers.md) (`store/xdbmemory`, `store/xdbfs`, `store/xdbredis`, `store/xdbsqlite`) are pure storage. They read tuples, apply mutations, and store definitions verbatim.

```
        Store facade   record + tuple + schema verbs, all writes compiled
                       to mutations; record assembly from tuples; _id
                       projection; namespace derivation; tx orchestration
  ┌──────────────────────┐
  │  logging             │  observes            (opt-in)
  │  schema enforcement  │  modes, Required, dynamic evolve,
  │                      │  Def.Validate, revision CAS, system-field
  │                      │  stamping             (ALWAYS installed)
  │  schema cache        │  accelerates         (opt-in)
  │  versioning          │  record CAS, _version/_updated stamping
  │                      │                       (ALWAYS installed)
  └──────────────────────┘
        Driver           pure storage: memory | fs | redis | sqlite
```

## Constructing a Store

Construct a `Store` with `store.New`. It installs schema enforcement and record versioning on every driver:

```go
// In-memory (reference, testing, embedded)
st := store.New(xdbmemory.NewDriver())

// Filesystem
d, err := xdbfs.NewDriver("/path/to/data", xdbfs.Options{})
st := store.New(d)

// Redis
st := store.New(xdbredis.NewDriver(client))

// SQLite
d, err := xdbsqlite.NewDriver(db)
st := store.New(d)
```

Options enable write logging and schema caching:

```go
st := store.New(d,
    store.WithLogger(logger),      // logs driver writes
    store.WithSchemaCache(),       // read-through cache for schema defs
)
```

Enforcement reads the schema of the record on every write. `WithSchemaCache` removes that driver round-trip for remote backends. The facade keeps the cache coherent across transactional writes and dynamic-mode evolutions.

## Interface Hierarchy

```
Store
├── Closer
├── RecordStore
├── SchemaStore
├── NamespaceReader
└── TupleStore
```

The operation tables below list the record, schema, and namespace verbs. `TupleStore` is the attribute-level surface:

```go
type TupleStore interface {
    GetTuple(ctx context.Context, uri *core.URI) (*core.Tuple, error)
    GetTuples(ctx context.Context, uris ...*core.URI) ([]*core.Tuple, error)
    PutTuples(ctx context.Context, tuples ...*core.Tuple) error
    DeleteTuples(ctx context.Context, uris ...*core.URI) error
}
```

## One Write Primitive

The table shows how each facade verb reaches the driver. Every write verb compiles to a mutation with explicit intent, one of the four ops in [Drivers](drivers.md). The two read verbs are a tuple scan and a point read:

| Facade verb          | Compiles to                                         |
| -------------------- | --------------------------------------------------- |
| `CreateRecord(r)`    | `{Path, OpCreate, r.Tuples()}`                      |
| `UpsertRecord(r)`    | `{Path, OpPut, r.Tuples()}`                         |
| `DeleteRecord(uri)`  | `{Path, OpDelete}`                                  |
| `PutTuples(ts...)`   | `{Path, OpPatch, ts}` per record path               |
| `DeleteTuples(uris)` | `{Path, OpDelete, Attrs}` per record path           |
| `GetRecord(uri)`     | tuple scan of the path -> assemble. Empty = NotFound |
| `GetTuple(uri)`      | point read -> absence = NotFound                     |

Record and patch behavior:

- Records are tuple sets. A record comes into existence when its first tuples are written. It ceases to exist when its last tuple is removed. A record with zero tuples cannot be represented, so a write of an empty record stores nothing.

- `PutTuples` patches the named attributes and preserves the others. `UpsertRecord` replaces the full record.

## Operation Behavior

### Record Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetRecord` | Returns record | `ErrNotFound` | Read by URI (ns + schema + id) |
| `ListRecords` | Returns page of records | Empty page (no error) | URI scope: ns-only or ns+schema |
| `CreateRecord` | `ErrAlreadyExists` | Creates record | Insert only. Rejects duplicates |
| `UpsertRecord` | Full replace | Creates record | Full replacement; checks `_version` when supplied |
| `DeleteRecord` | Deletes record | `ErrNotFound` | Remove by URI |

### Tuple Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetTuple` | Returns tuple | `ErrNotFound` | Read by attribute-level URI (`…#attr`) |
| `GetTuples` | Returns present tuples | Omitted (no error) | Batch point reads |
| `PutTuples` | Overwrites those attributes | Creates record | Patch. Other attributes untouched |
| `DeleteTuples` | Removes those attributes | No-op (idempotent) | Last tuple removed = record removed |

### Schema Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetSchema` | Returns definition | `ErrNotFound` | Read by URI (ns + schema) |
| `ListSchemas` | Returns page of definitions | Empty page (no error) | Optional namespace scope |
| `CreateSchema` | `ErrAlreadyExists` | Creates schema (revision 1) | Insert only |
| `UpdateSchema` | Replace + revision CAS | `ErrNotFound` | Mode, field types, array element types, `indexed`, and `unique` are immutable |
| `DeleteSchema` | Deletes schema | `ErrNotFound` | Remove by URI |
| `DeleteSchemaRecords` | Deletes all records | No-op if none exist | Record cleanup. The definition stays |

`UpdateSchema` applies an optimistic-concurrency check. A definition with `Revision` N updates only if the stored revision is N. A `Revision` of 0 means unconditional. On success the stored revision becomes N+1. Concurrent conflicting updates fail with `core.ErrConflict`.

### Namespace Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetNamespace` | Returns namespace | `ErrNotFound` | Read by URI (ns only) |
| `ListNamespaces` | Returns page of namespaces | Empty page (no error) | Lists all known namespaces |

Namespaces are derived from schemas. There is no writer interface. A namespace exists when at least one schema exists within it. The derivation lives in the facade. Drivers know nothing about namespaces.

## Enforcement

Middleware that `store.New` always installs enforces schema policy. The policy is uniform on every backend:

- Declared fields are type-checked on every write, in every schema [mode](schemas.md). Strict mode rejects undeclared attributes, flexible mode accepts them as-is, and dynamic mode infers them and evolves the schema.

- `Required` fields are checked on full-record writes, and on patches that create a record.

- `DeleteTuples` cannot remove a `Required` attribute from a record. A delete of the whole record is permitted.

- Schema updates validate compatibility (`ValidateUpdate`) and apply the revision CAS.

- Every definition is stamped with the `_version` and `_updated` system fields. A definition cannot declare a field name that starts with `_`. See [Versioning](versioning.md).

Violations are reported as `core.ErrSchemaViolation`, which wraps the specific schema error.

## Versioning

Every record carries `_id`, `_version`, and `_updated`. A write can echo `_version` back as an optimistic-concurrency precondition. This makes read-modify-write safe by default. A stale version fails with `core.ErrConflict`. See [Versioning](versioning.md) for the full contract.

## Optional Capabilities

| Interface         | Method                                                                                | Purpose                                            |
| ----------------- | ------------------------------------------------------------------------------------- | -------------------------------------------------- |
| `HealthChecker`   | `Health(ctx) error`                                                                   | Connectivity check (< 1 second)                    |
| `TX`              | `Run(ctx, func(tx Store) error) error`                                                | Transactional batch operations                     |
| `Validator`       | `ValidateRecord(ctx, record, op) error`, `ValidateDeleteRecord(ctx, uri) error`       | Validate a write or a delete without a write (dry run) |

The Store returned by `store.New` always implements `Validator`. It runs the same enforcement checks that a real write runs, without touching the driver. Dynamic-mode evolution is computed and discarded. The Store implements `TX` when the driver supports native transactions (memory, sqlite). On such stores, every write verb runs inside a transaction, so the enforcement checks and the write are atomic. Drivers without transactions (fs, redis) fall back to sequential execution.

```go
if tx, ok := st.(store.TX); ok {
    err := tx.Run(ctx, func(txs store.Store) error {
        // reads see writes made within the same transaction
        return txs.CreateRecord(ctx, record)
    })
}
```

## Querying and Pagination

List operations obey the [AIP-132](https://google.aip.dev/132) (List) and [AIP-160](https://google.aip.dev/160) (Filtering) patterns. They accept a `Query` and return a `Page`:

```go
type Query struct {
    URI    *core.URI // scope: ns-only or ns+schema
    Filter string    // CEL filter expression
    Fields []string  // not read by the facade or any driver
    Limit  int
    Offset int
}

type Page[T any] struct {
    Items      []T
    Total      int
    NextOffset int // 0 means no more pages
}
```

- `URI` determines the scope. An ns-only URI lists across all schemas. An ns+schema URI lists a single schema

- `Filter` is a [CEL expression](filters.md) evaluated against each record

- `Limit` defaults to 20, with a maximum of 1000

- `Offset` is zero-based

- `NextOffset` is 0 when there are no more pages

- `Total` is the total count of matching items, not only the current page

The facade synthesizes lists from tuple scans, filters in-process, and paginates. A driver with native filter pushdown handles schema-scoped queries in the database instead. The sqlite driver compiles CEL to SQL WHERE clauses. A field marked `indexed` or `unique` in its [schema](schemas.md) accelerates that pushdown on sqlite. The SQLite column engine creates indexes for these markers and rejects duplicate values on a `unique` field with `ErrUniqueViolation`. Other drivers retain the markers without creating indexes or enforcing uniqueness.

## Errors

| Error                       | Returned By                                                                        | Meaning                                                                    |
| --------------------------- | ---------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `core.ErrNotFound`          | Get, Update, Delete                                                                | Requested resource does not exist                                          |
| `core.ErrAlreadyExists`     | Create                                                                             | Resource already exists                                                    |
| `core.ErrSchemaViolation`   | `CreateRecord`, `UpsertRecord`, `PutTuples`, `DeleteTuples`, `CreateSchema`, `UpdateSchema` | Data or definition violates the schema                            |
| `core.ErrConflict`          | `UpdateSchema`, record writes                                                      | Revision or `_version` CAS failed. The base of the caller is stale         |
| `core.ErrUniqueViolation`   | `CreateRecord`, `UpsertRecord`, `PutTuples`                                        | A write collides with a `unique` field on another record. Only a backend with a unique index returns it |

All errors are sentinel values. Use `errors.Is(err, core.ErrNotFound)` to test for one.

The facade attributes batch tuple-write failures (`PutTuples`, `DeleteTuples`). A `store.MutationError` names the index and the record path of the failing mutation, and wraps the sentinel. Drivers return bare sentinels.

## Failure Modes

### Record and Tuple Operations

```
CreateRecord
├── OK                → record stored
├── ErrAlreadyExists  → record with same URI exists, nothing changed
├── ErrSchemaViolation→ payload has wrong types, undeclared attrs (strict), or missing required attrs
└── context error     → timeout or cancellation, record may or may not be stored

UpsertRecord
├── OK                → record stored (created or replaced)
├── ErrSchemaViolation→ data violates the schema, nothing changed
└── context error     → timeout or cancellation

DeleteRecord
├── OK                → record removed
├── ErrNotFound       → no record at this URI, nothing changed
└── context error     → timeout or cancellation

PutTuples
├── OK                → tuples patched (records created as needed)
├── ErrSchemaViolation→ type mismatch, undeclared attr (strict), or patch-that-creates
│                       missing required attrs
└── context error     → timeout or cancellation

DeleteTuples
├── OK                → tuples removed (no-op for absent ones)
├── ErrSchemaViolation→ attempted to delete a required attr
└── context error     → timeout or cancellation
```

### Schema Operations

```
CreateSchema
├── OK                → schema stored with revision 1
├── ErrAlreadyExists  → schema with same URI exists, nothing changed
├── ErrSchemaViolation→ malformed definition
└── context error     → timeout or cancellation

UpdateSchema
├── OK                → schema replaced, revision bumped
├── ErrNotFound       → no schema at this URI, nothing changed
├── ErrSchemaViolation→ mode, field type, element type, indexed, or unique change attempted
├── ErrConflict       → revision CAS failed
└── context error     → timeout or cancellation

DeleteSchema
├── OK                → schema removed
├── ErrNotFound       → no schema at this URI, nothing changed
└── context error     → timeout or cancellation

DeleteSchemaRecords
├── OK                → all records for schema removed (no-op if none exist)
└── context error     → timeout or cancellation
```

### Batch Operations

```
TX.Run
├── OK                → all operations committed atomically
├── fn returns error  → all changes rolled back, error propagated
├── store error       → all changes rolled back, store error returned
└── context error     → all changes rolled back
```

### Backend-Specific Failures

| Backend | Additional Failure Modes |
|---------|------------------------|
| Memory | None. All operations are in-process |
| Filesystem | `os.ErrPermission` (directory not writable), disk full |
| Redis | Connection refused, connection timeout, pool exhausted |
| SQLite | Database locked (concurrent access), disk full, corrupt database |

Backend-specific errors are not wrapped as store sentinels. They propagate as-is. The service layer maps them to RPC error codes.

## Drivers

The [driver contract](drivers.md) supports these storage layouts and capabilities:

| Driver | Storage | TX | Filter pushdown |
|--------|---------|----|-----------------|
| `xdbmemory` | Go maps | yes |: |
| `xdbfs` | JSON file per record |: |: |
| `xdbredis` | hash per record |: |: |
| `xdbsqlite` | column or KV tables per schema | yes | CEL -> SQL |

Wrap a driver returned by `NewDriver` with `store.New` to apply store policy.

## Config

The `store.backend` key in `~/.xdb/config.json` selects the backend. See [Configuration](config.md) for the keys and defaults.

## Shared Test Suites

The `tests/` package provides shared suites that pin store behavior. The record, schema, namespace, tuple, types, and version suites run against every backend through the facade, and check the shared semantics. The batch suite runs on drivers with native transactions (memory, sqlite). The mode and cascade suites test facade policy, so they run once, on the memory driver. The driver suite (`tests.NewDriverSuite`) pins the raw [driver contract](drivers.md).

## Related Concepts

- [Drivers](drivers.md): The storage contract that drivers implement

- [Records](records.md): Assembled views over tuples

- [Tuples](tuples.md): The unit of storage and addressing

- [Schemas](schemas.md): Structure definitions and modes

- [Namespaces](namespaces.md): Organizational grouping

- [Filters](filters.md): CEL-based record filtering for list operations

- [Encoding](encoding.md): How records are serialized for storage
