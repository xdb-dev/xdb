---
title: Records
description: Mutable, thread-safe groups of tuples that represent one entity.
package: core
---

# Records

A **Record** is the set of [Tuples](tuples.md) that share the same path
(Namespace + Schema + ID). A record adds no data of its own. It only groups
tuples. A record exists exactly when at least one tuple exists at its path.
Records are similar to objects, structs, or rows in a database. A record
usually represents one entity in your domain.

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

Unlike tuples, records are **mutable**. You can add, update, and remove attributes after creation. Records are also **thread-safe**. A read-write mutex protects concurrent access.

From the [CLI](../../cmd/xdb/cli/CONTEXT.md), you read and write records with `xdb records <action>`. Payloads are JSON. `create` writes a new record and fails with `ALREADY_EXISTS` if the record exists. `update` patches the record. `upsert` replaces the whole record.

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

A record is a developer-experience type, not a storage unit. On the write
side, a record is a builder. The [Store](stores.md) compiles `CreateRecord`
and `UpsertRecord` into tuple mutations. On the read side, a record is an
assembled view. `GetRecord` scans the tuples at the record path and groups
them. Storage [drivers](drivers.md) never see a `core.Record`.

Because a record is exactly its tuple set, a record with no tuples does not
exist. When you store an empty record, the store persists nothing. When you
delete the last tuple of a record with `DeleteTuples`, the store deletes the
record. System attributes do not keep a record alive. When the last user tuple
is deleted, the record and its metadata are deleted with it.

## Related Concepts

- [Tuples](tuples.md) — The building blocks of a record
- [Schemas](schemas.md) — Validation of the record structure
- [Stores](stores.md) — Persisting and querying records
- [Encoding](encoding.md) — JSON serialization
