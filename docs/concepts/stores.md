---
title: Stores
description: Storage interfaces and implementations for persisting records, schemas, and namespaces.
package: store, store/xdbmemory, store/xdbfs
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

## Optional Interfaces

Implementations may also satisfy:

| Interface         | Method                                              | Purpose                         |
| ----------------- | --------------------------------------------------- | ------------------------------- |
| `HealthChecker`   | `Health(ctx) error`                                 | Connectivity check (< 1 second) |
| `BatchExecutor`   | `ExecuteBatch(ctx, func(tx Store) error) error`     | Transactional batch operations  |

## Pagination

List operations accept a `ListQuery` and return a `Page`:

```go
type ListQuery struct {
    Limit  int
    Offset int
}

type Page[T any] struct {
    Items  []T
    Total  int
    Limit  int
    Offset int
}
```

## Errors

| Error                  | Meaning                                      |
| ---------------------- | -------------------------------------------- |
| `ErrNotFound`          | Requested resource does not exist             |
| `ErrAlreadyExists`     | Resource already exists (on create)           |
| `ErrSchemaViolation`   | Data violates the schema definition           |

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

### Redis Store (`xdbredis`) — Planned

**Package:** `store/xdbredis`

A Redis-backed store using RedisJSON for structured storage. Intended for production use with high throughput.

## Configuration

The store backend is configured via `~/.xdb/config.json`. See the [README](../../README.md) for configuration examples.

## Shared Test Suite

The `tests/` package provides shared test suites that validate any `Store` implementation against the expected behavior. All implementations must pass these suites.

## Related Concepts

- [Records](records.md) — The primary data stored
- [Schemas](schemas.md) — Structure definitions stored alongside records
- [Namespaces](namespaces.md) — Organizational grouping
- [Encoding](encoding.md) — How records are serialized for storage
