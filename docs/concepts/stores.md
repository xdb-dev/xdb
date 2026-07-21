---
title: Stores
description: The store facade, enforcement middleware, and drivers that persist records, schemas, and namespaces across backends.
package: store, store/xdbmemory, store/xdbfs, store/xdbredis, store/xdbsqlite
---

# Stores

A **Store** is the persistence layer in XDB. It reads and writes [Records](records.md), [Tuples](tuples.md), [Schemas](schemas.md), and [Namespaces](namespaces.md) with identical semantics on every backend.

The store is split into two layers:

- The **facade + middleware** (`store` package) own all policy: schema validation, mode enforcement, dynamic evolution, revision CAS, record assembly, filtering, pagination, namespace derivation, and transaction orchestration.
- **[Drivers](drivers.md)** (`store/xdbmemory`, `store/xdbfs`, `store/xdbredis`, `store/xdbsqlite`) are pure storage: they read tuples, apply mutations, and store schema definitions verbatim.

```
        Store facade   record + tuple + schema verbs, all writes compiled
                       to mutations; record assembly from tuples; namespace
                       derivation; tx orchestration
  ┌──────────────────────┐
  │  logging             │  observes            (opt-in)
  │  schema cache        │  accelerates         (opt-in)
  │  schema enforcement  │  modes, Required, dynamic evolve,
  │                      │  Def.Validate, revision CAS   (ALWAYS installed)
  └──────────────────────┘
        Driver           pure storage: memory | fs | redis | sqlite
```

## Constructing a Store

`store.New` is the **only** way to obtain a `Store`. It installs the schema enforcement middleware unconditionally — a Store that skips validation cannot be constructed:

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

Options add observability and acceleration around enforcement:

```go
st := store.New(d,
    store.WithLogger(logger),      // logs driver writes
    store.WithSchemaCache(),       // read-through cache for schema defs
)
```

`WithSchemaCache` removes a driver round-trip per write for remote backends (enforcement reads the record's schema on every write). The facade keeps the cache coherent across transactional writes and dynamic-mode evolutions.

## Interface Hierarchy

```
Store
├── Closer
├── RecordStore
├── SchemaStore
├── NamespaceReader
└── TupleStore
```

Record, schema, and namespace verbs keep their long-standing shapes (see the operation tables below). `TupleStore` is the attr-level surface:

```go
type TupleStore interface {
    GetTuple(ctx context.Context, uri *core.URI) (*core.Tuple, error)
    GetTuples(ctx context.Context, uris ...*core.URI) ([]*core.Tuple, error)
    PutTuples(ctx context.Context, tuples ...*core.Tuple) error
    DeleteTuples(ctx context.Context, uris ...*core.URI) error
}
```

## One Write Primitive

Every write verb on the facade compiles to a mutation with explicit intent — the four-op table in [Drivers](drivers.md):

| Facade verb          | Compiles to                                     |
| -------------------- | ----------------------------------------------- |
| `CreateRecord(r)`    | `{Path, OpCreate, r.Tuples()}`                  |
| `UpsertRecord(r)`    | `{Path, OpPut, r.Tuples()}`                 |
| `DeleteRecord(uri)`  | `{Path, OpDelete}`                              |
| `PutTuples(ts...)`   | `{Path, OpPatch, ts}` per record path           |
| `DeleteTuples(uris)` | `{Path, OpDelete, Attrs}` per record path       |
| `GetRecord(uri)`     | tuple scan of the path → assemble; empty = NotFound |
| `GetTuple(uri)`      | point read → absence = NotFound                 |

Two consequences worth knowing:

- **Records are tuple sets.** A record springs into existence when its first tuples are written and ceases to exist when its last tuple is removed. A record with zero tuples is unrepresentable — creating an empty record stores nothing.
- **`PutTuples` is a merge** (patch), not a replace: other attrs of the record are untouched. `UpsertRecord` is a full replace (put).

## Operation Behavior

### Record Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetRecord` | Returns record | `ErrNotFound` | Read by URI (ns + schema + id) |
| `ListRecords` | Returns page of records | Empty page (no error) | URI scope: ns-only or ns+schema |
| `CreateRecord` | `ErrAlreadyExists` | Creates record | Insert only — rejects duplicates |
| `UpsertRecord` | Full replace | Creates record | Unconditional write (put) — always succeeds |
| `DeleteRecord` | Deletes record | `ErrNotFound` | Remove by URI |

### Tuple Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetTuple` | Returns tuple | `ErrNotFound` | Read by attr-level URI (`…#attr`) |
| `GetTuples` | Returns present tuples | Omitted (no error) | Batch point reads |
| `PutTuples` | Overwrites those attrs | Creates record | Merge — other attrs untouched |
| `DeleteTuples` | Removes those attrs | No-op (idempotent) | Last tuple removed = record removed |

### Schema Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetSchema` | Returns schema def | `ErrNotFound` | Read by URI (ns + schema) |
| `ListSchemas` | Returns page of schemas | Empty page (no error) | Optional namespace scope |
| `CreateSchema` | `ErrAlreadyExists` | Creates schema (revision 1) | Insert only |
| `UpdateSchema` | Replace + revision CAS | `ErrNotFound` | Mode and field types are immutable |
| `DeleteSchema` | Deletes schema | `ErrNotFound` | Remove by URI |
| `DeleteSchemaRecords` | Deletes all records | No-op if none exist | Record cleanup; the schema def stays |

`UpdateSchema` applies an optimistic-concurrency check: a def with `Revision` N updates only if the stored revision is N (0 means unconditional). On success the stored revision becomes N+1. Concurrent conflicting updates fail with `core.ErrConflict`.

### Namespace Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetNamespace` | Returns namespace | `ErrNotFound` | Read by URI (ns only) |
| `ListNamespaces` | Returns page of namespaces | Empty page (no error) | Lists all known namespaces |

Namespaces are derived from schemas — there is no writer interface. A namespace exists when at least one schema exists within it. The derivation lives in the facade; drivers know nothing about namespaces.

## Enforcement

Schema policy is enforced by middleware that `store.New` always installs — **uniformly on every backend**:

- Declared fields are type-checked on every write, per the schema [mode](schemas.md): strict rejects undeclared attrs, flexible ignores them, dynamic infers and evolves the schema.
- `Required` fields are checked on full-record writes, and on merges that create a record.
- `DeleteTuples` cannot strip a `Required` attr from a record (deleting the whole record is fine).
- Schema updates validate compatibility (`ValidateUpdate`) and apply the revision CAS.

Violations are reported as `core.ErrSchemaViolation` (wrapping the specific schema error).

## Optional Capabilities

| Interface         | Method                                              | Purpose                         |
| ----------------- | --------------------------------------------------- | ------------------------------- |
| `HealthChecker`   | `Health(ctx) error`                                 | Connectivity check (< 1 second) |
| `TX`              | `Run(ctx, func(tx Store) error) error`              | Transactional batch operations  |

The Store returned by `store.New` implements `TX` when the driver supports native transactions (memory, sqlite). On such stores, **every** write verb runs inside a transaction, so enforcement checks and the write are atomic. Drivers without transactions (fs, redis) fall back to sequential execution.

```go
if tx, ok := st.(store.TX); ok {
    err := tx.Run(ctx, func(txs store.Store) error {
        // reads see writes made within the same transaction
        return txs.CreateRecord(ctx, record)
    })
}
```

## Querying and Pagination

List operations follow [AIP-132](https://google.aip.dev/132) (List) and [AIP-160](https://google.aip.dev/160) (Filtering) patterns. They accept a `Query` and return a `Page`:

```go
type Query struct {
    URI    *core.URI // scope: ns-only or ns+schema
    Filter string    // CEL filter expression
    Fields []string  // field mask (not yet implemented)
    Limit  int
    Offset int
}

type Page[T any] struct {
    Items      []T
    Total      int
    NextOffset int // 0 means no more pages
}
```

- `URI` determines the scope — ns-only lists across all schemas, ns+schema lists a single schema
- `Filter` is a [CEL expression](filters.md) evaluated against each record
- `Limit` defaults to 20, max 1000
- `Offset` is zero-based
- `NextOffset` is 0 when there are no more pages
- `Total` is the total count of matching items (not just the current page)

The facade synthesizes lists from tuple scans, filters in-process, and paginates. Drivers with native filter pushdown (sqlite compiles CEL to SQL WHERE clauses) handle schema-scoped queries in the database instead.

## Errors

| Error                  | Returned By | Meaning                                      |
| ---------------------- | ----------- | -------------------------------------------- |
| `core.ErrNotFound`          | Get, Update, Delete | Requested resource does not exist    |
| `core.ErrAlreadyExists`     | Create | Resource already exists                       |
| `core.ErrSchemaViolation`   | Create, Update, Upsert, PutTuples | Data violates the schema definition |
| `core.ErrConflict`          | UpdateSchema | Revision CAS failed — the caller's base is stale |

All errors are sentinel values — use `errors.Is(err, core.ErrNotFound)` to check. (The `store.Err*` aliases are deprecated re-exports of the `core` sentinels.)

Batch tuple-write failures (`PutTuples`, `DeleteTuples`) are attributed by the facade: a `store.MutationError` names the failing mutation's index and record path and wraps the sentinel. Drivers themselves return bare sentinels.

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
├── OK                → tuples merged (records created as needed)
├── ErrSchemaViolation→ type mismatch, undeclared attr (strict), or merge-that-creates
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
├── ErrSchemaViolation→ mode change or field type change attempted
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
| Memory | None — all operations are in-process |
| Filesystem | `os.ErrPermission` (directory not writable), disk full |
| Redis | Connection refused, connection timeout, pool exhausted |
| SQLite | Database locked (concurrent access), disk full, corrupt database |

Backend-specific errors are not wrapped as store sentinel errors — they propagate as-is. The service layer maps them to appropriate RPC error codes.

## Backends

See [Drivers](drivers.md) for the contract backends implement and how each maps tuples onto its storage. In brief:

| Driver | Storage | TX | Filter pushdown |
|--------|---------|----|-----------------|
| `xdbmemory` | Go maps | yes | — |
| `xdbfs` | JSON file per record | — | — |
| `xdbredis` | hash per record | — | — |
| `xdbsqlite` | column or KV tables per schema | yes | CEL → SQL |

## Migrating from pre-driver constructors

Driver packages export Drivers, not Stores — deliberately, so a Store that skips enforcement cannot be built:

| Before | After |
| ------ | ----- |
| `xdbmemory.New()` | `store.New(xdbmemory.NewDriver())` |
| `xdbfs.New(root, opts)` | `d, err := xdbfs.NewDriver(root, opts)` then `store.New(d)` |
| `xdbredis.New(client, opts...)` | `store.New(xdbredis.NewDriver(client, opts...))` |
| `xdbsqlite.New(db, opts...)` | `d, err := xdbsqlite.NewDriver(db, opts...)` then `store.New(d)` |

## Configuration

The store backend is configured via `~/.xdb/config.json`. See the [README](../../README.md) for configuration examples.

## Shared Test Suites

The `tests/` package provides shared suites that pin store behavior. The record/schema/namespace/mode/cascade/tuple suites run against every backend **through the facade**, proving identical semantics; the driver suite (`tests.NewDriverSuite`) pins the raw [driver contract](drivers.md).

## Related Concepts

- [Drivers](drivers.md) — The storage contract backends implement
- [Records](records.md) — Assembled views over tuples
- [Tuples](tuples.md) — The unit of storage and addressing
- [Schemas](schemas.md) — Structure definitions and modes
- [Namespaces](namespaces.md) — Organizational grouping
- [Filters](filters.md) — CEL-based record filtering for list operations
- [Encoding](encoding.md) — How records are serialized for storage
