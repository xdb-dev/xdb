---
title: Stores
description: Storage interfaces and implementations for persisting records, schemas, and namespaces.
package: store, store/xdbmemory, store/xdbfs, store/xdbredis, store/xdbsqlite
---

# Stores

A **Store** is the persistence layer in XDB. The store package defines interfaces for reading and writing [Records](records.md), [Schemas](schemas.md), and [Namespaces](namespaces.md). Multiple implementations allow the same data model to work across different storage backends.

## Interface Hierarchy

The store interfaces are layered and composable:

```
Store
├── RecordStore
│   ├── RecordReader
│   └── RecordWriter
├── SchemaStore
│   ├── SchemaReader
│   └── SchemaWriter
└── NamespaceReader
```

### RecordStore

```go
type RecordReader interface {
    GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error)
    ListRecords(ctx context.Context, uri *core.URI, q *ListQuery) (*Page[*core.Record], error)
}

type RecordWriter interface {
    CreateRecord(ctx context.Context, record *core.Record) error
    UpdateRecord(ctx context.Context, record *core.Record) error
    UpsertRecord(ctx context.Context, record *core.Record) error
    DeleteRecord(ctx context.Context, uri *core.URI) error
}
```

### SchemaStore

```go
type SchemaReader interface {
    GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error)
    ListSchemas(ctx context.Context, uri *core.URI, q *ListQuery) (*Page[*schema.Def], error)
}

type SchemaWriter interface {
    CreateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error
    UpdateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error
    DeleteSchema(ctx context.Context, uri *core.URI) error
}
```

### NamespaceReader

```go
type NamespaceReader interface {
    GetNamespace(ctx context.Context, uri *core.URI) (*core.NS, error)
    ListNamespaces(ctx context.Context, q *ListQuery) (*Page[*core.NS], error)
}
```

### Full Store

```go
type Store interface {
    RecordStore
    SchemaStore
    NamespaceReader
}
```

## Operation Behavior

### Record Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetRecord` | Returns record | `ErrNotFound` | Read by URI (ns + schema + id) |
| `ListRecords` | Returns page of records | Empty page (no error) | URI scope: ns-only or ns+schema |
| `CreateRecord` | `ErrAlreadyExists` | Creates record | Insert only — rejects duplicates |
| `UpdateRecord` | Full replace | `ErrNotFound` | Replaces the entire record (all tuples) |
| `UpsertRecord` | Full replace | Creates record | Unconditional write — always succeeds |
| `DeleteRecord` | Deletes record | `ErrNotFound` | Remove by URI |

**Current update semantics:** All store implementations treat `UpdateRecord` and `UpsertRecord` as **full replacement** — the caller provides a complete `*core.Record` and the entire stored record is replaced. No partial merge occurs at the store level.

The service layer (planned) will implement **patch semantics** on top of `UpdateRecord` by reading the existing record, merging supplied fields, and writing back the full record. See the [CLI plan](../plans/2026-03-09-agent-first-cli.md) for details.

**How full replacement works in each backend:**

- **Memory:** Direct map assignment (`records[key] = record`)
- **Filesystem:** Overwrites the JSON file
- **Redis:** Deletes the hash key, then writes all fields
- **SQLite (KV):** Deletes all rows for the ID, then inserts new rows
- **SQLite (column):** `UPDATE ... SET col1=?, col2=?, ...` with nil for missing columns

### Schema Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetSchema` | Returns schema def | `ErrNotFound` | Read by URI (ns + schema) |
| `ListSchemas` | Returns page of schemas | Empty page (no error) | Optional namespace scope |
| `CreateSchema` | `ErrAlreadyExists` | Creates schema | Insert only — rejects duplicates |
| `UpdateSchema` | Full replace | `ErrNotFound` | Replaces the schema definition |
| `DeleteSchema` | Deletes schema | `ErrNotFound` | Remove by URI |

### Namespace Operations

| Operation | Exists | Not Exists | Semantics |
|-----------|--------|------------|-----------|
| `GetNamespace` | Returns namespace | `ErrNotFound` | Read by URI (ns only) |
| `ListNamespaces` | Returns page of namespaces | Empty page (no error) | Lists all known namespaces |

Namespaces are derived from schemas — there is no writer interface. A namespace exists when at least one schema exists within it.

## Optional Interfaces

Implementations may also satisfy:

| Interface         | Method                                              | Purpose                         |
| ----------------- | --------------------------------------------------- | ------------------------------- |
| `HealthChecker`   | `Health(ctx) error`                                 | Connectivity check (< 1 second) |
| `BatchExecutor`   | `ExecuteBatch(ctx, func(tx Store) error) error`     | Transactional batch operations  |

`BatchExecutor` runs a function within a transaction. If the function returns an error, all changes are rolled back. The `tx Store` passed to the function is a transactional view — reads see writes made within the same transaction.

## Pagination

List operations accept a `ListQuery` and return a `Page`:

```go
type ListQuery struct {
    Filter string
    Limit  int
    Offset int
}

type Page[T any] struct {
    Items      []T
    Total      int
    NextOffset int // 0 means no more pages
}
```

- `Limit` defaults to 20, max 1000
- `Offset` is zero-based
- `NextOffset` is 0 when there are no more pages
- `Total` is the total count of matching items (not just the current page)
- `Filter` is a raw filter string (parsed by the service layer)

## Errors

| Error                  | Returned By | Meaning                                      |
| ---------------------- | ----------- | -------------------------------------------- |
| `ErrNotFound`          | Get, Update, Delete | Requested resource does not exist    |
| `ErrAlreadyExists`     | Create | Resource already exists                       |
| `ErrSchemaViolation`   | Create, Update, Upsert | Data violates the schema definition |

All errors are sentinel values — use `errors.Is(err, store.ErrNotFound)` to check.

> **Note:** `ErrSchemaViolation` is currently only enforced by schema-aware backends (xdbsqlite). Other backends (xdbmemory, xdbfs, xdbredis) store data without schema validation — the service layer will enforce validation uniformly across all backends.

## Failure Modes

> `ErrSchemaViolation` paths below only apply to schema-aware backends (currently xdbsqlite). Other backends skip validation.

### Record Operations

```
CreateRecord
├── OK                → record stored
├── ErrAlreadyExists  → record with same URI exists, nothing changed
├── ErrSchemaViolation→ strict schema active, payload has wrong types or missing required attrs
└── context error     → timeout or cancellation, record may or may not be stored

UpdateRecord
├── OK                → record fully replaced
├── ErrNotFound       → no record at this URI, nothing changed
├── ErrSchemaViolation→ new data violates strict schema, original record unchanged
└── context error     → timeout or cancellation, record may be partially written (backend-dependent)

UpsertRecord
├── OK                → record stored (created or replaced)
├── ErrSchemaViolation→ data violates strict schema, nothing changed
└── context error     → timeout or cancellation

DeleteRecord
├── OK                → record removed
├── ErrNotFound       → no record at this URI, nothing changed
└── context error     → timeout or cancellation

GetRecord
├── OK                → record returned
├── ErrNotFound       → no record at this URI
└── context error     → timeout or cancellation

ListRecords
├── OK                → page returned (may be empty — empty page is not an error)
└── context error     → timeout or cancellation
```

### Schema Operations

```
CreateSchema
├── OK                → schema stored
├── ErrAlreadyExists  → schema with same URI exists, nothing changed
└── context error     → timeout or cancellation

UpdateSchema
├── OK                → schema definition replaced
├── ErrNotFound       → no schema at this URI, nothing changed
├── ErrSchemaViolation→ mode change or field type change attempted
└── context error     → timeout or cancellation

DeleteSchema
├── OK                → schema removed
├── ErrNotFound       → no schema at this URI, nothing changed
└── context error     → timeout or cancellation
```

### Batch Operations

```
ExecuteBatch
├── OK                → all operations committed atomically
├── fn returns error  → all changes rolled back, error propagated
├── store error       → all changes rolled back, store error returned
└── context error     → all changes rolled back
```

If a store does not implement `BatchExecutor`, the service layer executes operations sequentially without transaction guarantees.

### Backend-Specific Failures

| Backend | Additional Failure Modes |
|---------|------------------------|
| Memory | None — all operations are in-process |
| Filesystem | `os.ErrPermission` (directory not writable), disk full |
| Redis | Connection refused, connection timeout, pool exhausted |
| SQLite | Database locked (concurrent access), disk full, corrupt database |

Backend-specific errors are not wrapped as store sentinel errors — they propagate as-is. The service layer maps them to appropriate RPC error codes.

## Implementations

### In-Memory Store (`xdbmemory`)

**Package:** `store/xdbmemory`

A reference implementation that holds all data in Go maps. Useful for testing and embedded use.

```go
store := xdbmemory.New()
```

- All state is in memory — lost on process exit.
- Thread-safe via `sync.RWMutex`.
- Health check always succeeds.
- Implements `BatchExecutor` via snapshot-and-rollback.

### Filesystem Store (`xdbfs`)

**Package:** `store/xdbfs`

Persists data as JSON files on the local filesystem. Good for development, CLI tools, and local workflows.

```go
store, err := xdbfs.New("/path/to/data", xdbfs.Options{
    Indent:      "  ",     // Pretty-print JSON (default)
    CompactJSON: false,    // Set true for compact output
})
```

**Directory layout:**

```
<root>/
├── <namespace>/
│   ├── <schema>/
│   │   ├── _schema.json    # Schema definition
│   │   ├── <id>.json       # Record files
│   │   └── <id>.json
│   └── <schema>/
└── <namespace>/
```

- Thread-safe via `sync.RWMutex`.
- Atomic writes (temp file + rename).
- Health check verifies the root directory exists and is writable.
- JSON files are human-readable and inspectable.

### Redis Store (`xdbredis`)

**Package:** `store/xdbredis`

A Redis-backed store using hashes for structured storage. Suitable for production use with high throughput.

```go
store, err := xdbredis.New(ctx, xdbredis.Options{
    Addr: "localhost:6379",
})
```

- Records stored as Redis hashes, one hash per record.
- Schemas stored as JSON strings in Redis keys.
- Namespaces derived from key prefixes.
- Uses transaction pipelines for atomic individual operations.
- Health check uses `PING`.
- Requires a running Redis instance (`make services-up`).

### SQLite Store (`xdbsqlite`)

**Package:** `store/xdbsqlite`

A SQLite-backed store with two storage strategies based on schema mode.

```go
db, err := sql.Open("sqlite3", "xdb.db")
store, err := xdbsqlite.New(db)
```

- **Column strategy** (strict schemas): One SQL column per schema attribute. Type-safe, indexed queries.
- **KV strategy** (flexible schemas): Key-value rows per record. Accommodates any attribute set.
- Implements `BatchExecutor` via SQLite transactions.
- Health check uses `PRAGMA quick_check`.
- Single-file database — easy to back up and move.

## Configuration

The store backend is configured via `~/.xdb/config.json`. See the [README](../../README.md) for configuration examples.

## Shared Test Suite

The `tests/` package provides shared test suites that validate any `Store` implementation against the expected behavior. All implementations must pass these suites.

## Related Concepts

- [Records](records.md) — The primary data stored
- [Schemas](schemas.md) — Structure definitions stored alongside records
- [Namespaces](namespaces.md) — Organizational grouping
- [Encoding](encoding.md) — How records are serialized for storage
