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

## Thread Safety

Records use `sync.RWMutex` internally:

- `Set()` acquires a write lock
- `Get()`, `Tuples()`, `IsEmpty()` acquire a read lock

This makes records safe to use from multiple goroutines without external synchronization.

## Related Concepts

- [Tuples](tuples.md) — The building blocks of a record
- [Schemas](schemas.md) — Validating record structure
- [Stores](stores.md) — Persisting and querying records
- [Encoding](encoding.md) — JSON serialization
