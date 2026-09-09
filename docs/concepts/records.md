---
title: Records
description: Mutable, thread-safe groups of tuples that represent one entity.
package: core
---

# Records

A record groups the [tuples](tuples.md) that share a path (namespace, schema, and ID). It usually describes one domain entity. A record exists while its path has user tuples; it has no separate stored container.

## Structure

```
┌───────────────────────────────────────────┐
│  Record                                   │
│  Path: xdb://com.example/posts/post-123   │
├───────────────────────────────────────────┤
│  title   → "Hello World"    (STRING)      │
│  author  → "user-001"       (STRING)      │
│  views   → 42               (INTEGER)     │
│  draft   → false            (BOOLEAN)     │
└───────────────────────────────────────────┘
```

Records are mutable: you can add, update, and remove attributes. A read-write mutex protects concurrent access.

Use `xdb records <action>` to read and write JSON payloads through the [CLI](../../cmd/xdb/cli/CONTEXT.md). `create` returns the existing record for an identical payload and reports `CONFLICT` for different data at the same URI. `update` patches a record; `upsert` replaces it.

## Creating Records

```go
record := core.NewRecord("com.example", "posts", "post-123")
```

`NewRecord` panics on an invalid namespace, schema, or ID.

## Setting Attributes

The `Set` method adds or updates a tuple in the record. Calls to `Set` can be chained:

```go
record := core.NewRecord("com.example", "posts", "post-123").
    Set("title", "Hello World").
    Set("author", "user-001").
    Set("views", 42).
    Set("draft", false).
    Set("created", time.Now())
```

`Set` accepts any value that the [Type](types.md) system supports. XDB wraps the value in a `*Value` with the correct type.

## Reading Attributes

### Get a Single Tuple

```go
tuple := record.Get("title")
if tuple != nil {
    title, err := tuple.AsStr()
}
```

`Get` returns `nil` if the attribute does not exist. The `As*` accessors are
safe to call on that nil. `record.Get("title").AsStr()` returns the zero
value and [`ErrAttrNotFound`](../../core/errors.go) when the attribute is
absent. This makes a typo different from an empty value.

### Get All Tuples

```go
tuples := record.Tuples() // returns []*Tuple (copy)
```

`Tuples()` returns a copy of the internal tuple slice. Later `Set` calls do not change the returned slice.

## Record Metadata

```go
record.URI()              // *URI   — full record URI (xdb://com.example/posts/post-123)
record.URI().SchemaURI()  // *URI   — schema URI (xdb://com.example/posts)
record.URI().NS()         // string — namespace
record.URI().Schema()     // string — schema name
record.URI().ID()         // string — record identifier
record.IsEmpty()          // bool   — true if the record has no tuples
```

### System Attributes

A record read from a [Store](stores.md) also carries three system attributes:

```go
record.Get("_id").AsStr()        // "post-123", projected from the path
record.Get("_version").AsInt()   // 3, incremented on every write
record.Get("_updated").AsTime()  // timestamp of the last write
```

When you write the record back, the store uses `_version` as an
optimistic-concurrency precondition. A mismatch returns `core.ErrConflict`.
As a result, read-modify-write is safe by default. See
[Versioning](versioning.md).

## Thread Safety

Records use `sync.RWMutex` internally:

- `Set()` takes a write lock

- `Get()`, `Tuples()`, and `IsEmpty()` take a read lock

As a result, records are safe to use from multiple goroutines without external synchronization.

## Records and Storage

Use records to build writes and read grouped tuples in Go. The [store](stores.md) converts `CreateRecord` and `UpsertRecord` calls into tuple mutations. `GetRecord` scans and groups tuples at the record path. Storage [drivers](drivers.md) operate on those tuples.

Writing an empty record stores nothing. Deleting the last user tuple with `DeleteTuples` removes the record and its system metadata.

## Related Concepts

- [Tuples](tuples.md): The building blocks of a record

- [Schemas](schemas.md): Validation of the record structure

- [Stores](stores.md): Persisting and querying records

- [Encoding](encoding.md): JSON serialization
