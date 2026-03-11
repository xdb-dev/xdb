---
title: Records
description: Mutable, thread-safe groups of tuples that represent a single entity.
package: core
---

# Records

A **Record** is a group of [Tuples](tuples.md) that share the same path (Namespace + Schema + ID). Records are similar to objects, structs, or rows in a database. They typically represent a single entity in your domain.

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

## Creating Records

```go
record := core.NewRecord("com.example", "posts", "post-123")
```

### Using the Builder

```go
record := core.New().
    NS("com.example").
    Schema("posts").
    ID("post-123").
    MustRecord()
```

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

`Get` returns `nil` if the attribute does not exist.

### Get All Tuples

```go
tuples := record.Tuples() // returns []*Tuple (copy)
```

`Tuples()` returns a copy of the internal tuple slice, safe to iterate without holding the lock.

## Record Metadata

```go
record.URI()       // *URI — full record URI (xdb://com.example/posts/post-123)
record.SchemaURI() // *URI — schema URI (xdb://com.example/posts)
record.NS()        // *NS
record.Schema()    // *Schema
record.ID()        // *ID
record.IsEmpty()   // bool — true if no tuples
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
