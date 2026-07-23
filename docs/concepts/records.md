---
title: Records
description: Mutable, thread-safe groups of tuples that represent a single entity.
package: core
---

# Records

A **Record** _is_ the set of [Tuples](tuples.md) that share the same path
(Namespace + Schema + ID) — it adds no data of its own, it groups tuples. A
record exists exactly when at least one tuple exists at its path. Records are
similar to objects, structs, or rows in a database, and typically represent a
single entity in your domain.

## Structure

```
┌──────────────────────────────────────────────┐
│               Record                          │
│  Path: xdb://com.example/posts/post-123       │
├──────────────────────────────────────────────┤
│  title    → "Hello World"       (STRING)      │
│  author   → "user-001"         (STRING)      │
│  views    → 42                  (INTEGER)     │
│  draft    → false               (BOOLEAN)     │
└──────────────────────────────────────────────┘
```

Unlike tuples, records are **mutable** — you can add, update, and remove attributes after creation. Records are also **thread-safe**, using a read-write mutex for concurrent access.

From the [CLI](../../cmd/xdb/cli/CONTEXT.md), records are what you read and write via `xdb records <action>`. Payloads are JSON; `create` is idempotent insert, `update` is patch merge, `upsert` is full replace.

## Creating Records

```go
record := core.NewRecord("com.example", "posts", "post-123")
```

`NewRecord` panics on an invalid namespace, schema, or ID.

## Setting Attributes

The `Set` method adds or updates a tuple in the record. It is chainable:

```go
record := core.NewRecord("com.example", "posts", "post-123").
    Set("title", "Hello World").
    Set("author", "user-001").
    Set("views", 42).
    Set("draft", false).
    Set("created", time.Now())
```

`Set` accepts any value that the [Type](types.md) system supports. The value is automatically wrapped in a `*Value` with the correct type.

## Reading Attributes

### Get a Single Tuple

```go
tuple := record.Get("title")
if tuple != nil {
    title, err := tuple.AsStr()
}
```

`Get` returns `nil` if the attribute does not exist. The `As*` accessors are
safe to chain on that nil: `record.Get("title").AsStr()` returns the zero
value and [`ErrAttrNotFound`](../../core/errors.go) when the attribute is
absent, so a typo is distinguishable from an empty value.

### Get All Tuples

```go
tuples := record.Tuples() // returns []*Tuple (copy)
```

`Tuples()` returns a copy of the internal tuple slice, safe to iterate without holding the lock.

## Record Metadata

```go
record.URI()              // *URI   — full record URI (xdb://com.example/posts/post-123)
record.URI().SchemaURI()  // *URI   — schema URI (xdb://com.example/posts)
record.URI().NS()         // string — namespace
record.URI().Schema()     // string — schema name
record.URI().ID()         // string — record identifier
record.IsEmpty()          // bool   — true if no tuples
```

### System Fields

A record read from a [Store](stores.md) also carries three system fields:

```go
record.Get("_id").AsStr()        // "post-123" — projected from the path
record.Get("_version").AsInt()   // 3 — bumped on every write
record.Get("_updated").AsTime()  // last write timestamp
```

Writing the record back uses `_version` as an optimistic-concurrency
precondition, so read-modify-write is safe by default. See
[Versioning](versioning.md).

## Thread Safety

Records use `sync.RWMutex` internally:

- `Set()` acquires a write lock
- `Get()`, `Tuples()`, `IsEmpty()` acquire a read lock

This makes records safe to use from multiple goroutines without external synchronization.

## Records and Storage

A record is a developer-experience type, not a storage unit. On the
write side it is a builder — the [Store](stores.md) compiles
`CreateRecord`/`UpsertRecord` into tuple mutations. On the read side it
is an assembled view — `GetRecord` scans the record path's tuples and
groups them. Storage [drivers](drivers.md) never see a `core.Record` at
all.

Because a record is exactly its tuple set, a record with no tuples
does not exist: storing an empty record persists nothing, and deleting
a record's last tuple (via `DeleteTuples`) removes the record. System
fields do not keep a record alive — removing the last *user* tuple
removes the record and its metadata with it.

## Related Concepts

- [Tuples](tuples.md) — The building blocks of a record
- [Schemas](schemas.md) — Validating record structure
- [Stores](stores.md) — Persisting and querying records
- [Encoding](encoding.md) — JSON serialization
