# In-Memory Store Reference Implementation

**Date:** 2026-03-10
**Status:** Draft
**Depends on:** `store/` package interfaces

## Overview

Implement `store/memstore`, an in-memory `store.Store` that serves as the reference implementation. It is the first backend, used for testing, embedded mode, and validating the store interface design.

## Goals

- Implement every method in `store.Store` (records, schemas, namespaces)
- Implement `store.HealthChecker` and `store.BatchExecutor`
- Thread-safe via `sync.RWMutex`
- No external dependencies beyond `core` and `store`
- Full test coverage using TDD

## Package

```
store/memstore/
  memstore.go       # Store struct, constructor
  records.go        # RecordReader + RecordWriter
  schemas.go        # SchemaReader + SchemaWriter
  namespaces.go     # NamespaceReader (derived from schemas)
  batch.go          # BatchExecutor (snapshot + rollback)
  memstore_test.go  # Tests for all operations
```

## Data Model

All state lives in maps protected by a single `sync.RWMutex`:

```go
type Store struct {
    mu      sync.RWMutex
    records map[string]*core.Record   // key: URI.Path() (ns/schema/id)
    schemas map[string]*store.SchemaDef // key: URI.Path() (ns/schema)
}
```

Namespaces are derived — `ListNamespaces` scans the schemas map and collects distinct NS values. `GetNamespace` checks if any schema exists in the given namespace.

## Implementation Phases

### Phase 1: Struct and Records

1. **`memstore.go`** — `Store` struct, `New()` constructor, `Health()` method
2. **`records.go`** — Implement `RecordReader` and `RecordWriter`:
   - `CreateRecord` — check key doesn't exist, insert
   - `GetRecord` — lookup by `URI.Path()`, return `ErrNotFound` if missing
   - `UpdateRecord` — check key exists, replace
   - `UpsertRecord` — insert or replace unconditionally
   - `DeleteRecord` — check key exists, delete
   - `ListRecords` — filter by URI scope (ns or ns/schema), apply `ListQuery` offset/limit, compute total

### Phase 2: Schemas

3. **`schemas.go`** — Implement `SchemaReader` and `SchemaWriter`:
   - Same CRUD pattern as records, keyed by schema URI path (ns/schema)
   - `ListSchemas` — if URI has only ns, list all schemas in that namespace; if nil, list all

### Phase 3: Namespaces

4. **`namespaces.go`** — Implement `NamespaceReader`:
   - `ListNamespaces` — scan schemas map, collect unique NS values, paginate
   - `GetNamespace` — scan schemas for any with matching NS, return `ErrNotFound` if none

### Phase 4: Batch

5. **`batch.go`** — Implement `BatchExecutor`:
   - `ExecuteBatch` — snapshot current maps, run `fn` against a transactional `Store`, on error restore snapshot
   - The transactional store is a shallow copy that writes to the same maps (under the same lock), but can be rolled back

## Key Design Decisions

| Decision | Rationale |
|---|---|
| Single `sync.RWMutex` | Simple, correct. Memstore is not a production database — optimize for clarity |
| Maps keyed by `URI.Path()` | Deterministic string key, avoids pointer equality issues |
| Namespaces derived from schemas | No separate namespace storage — consistent with the store interface having no `NamespaceWriter` |
| Snapshot-based batch | Deep-copy maps before `fn`, restore on error. Simple rollback without WAL complexity |
| `ListRecords` URI scoping | NS-only URI lists all records in namespace; NS+Schema URI lists records for that schema. Matches RPC `records.list` semantics |

## Test Plan

Tests follow TDD — write each test before the implementation.

### Records

- Create a record, get it back
- Create duplicate returns `ErrAlreadyExists`
- Get non-existent returns `ErrNotFound`
- Update existing record
- Update non-existent returns `ErrNotFound`
- Upsert creates when missing
- Upsert updates when present
- Delete existing record
- Delete non-existent returns `ErrNotFound`
- List records scoped by namespace
- List records scoped by namespace + schema
- List with offset and limit (pagination)
- List returns correct total count

### Schemas

- Create, get, update, delete (same pattern as records)
- List scoped by namespace
- List all schemas (nil URI)
- Duplicate create returns `ErrAlreadyExists`

### Namespaces

- List returns unique namespaces derived from schemas
- Get existing namespace
- Get non-existent namespace returns `ErrNotFound`
- Adding schema in new namespace makes it appear in list
- Deleting all schemas in a namespace removes it from list

### Batch

- Successful batch commits all changes
- Failed batch rolls back all changes
- Batch is atomic — partial failures don't persist

### Concurrency

- Concurrent reads don't block each other
- Concurrent writes are serialized correctly
- Read during write sees consistent state
